package scripts

import (
	"context"
	"fmt"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"google.golang.org/protobuf/types/known/timestamppb"
	"os"
	"pgk-schedule-script/utils"
	"regexp"
	"strconv"
	"strings"
	"time"
)
import "pgk-schedule-script/gen/go"

const (
	CredentialsFileGoogleSheet = "creds.json"
	SpreadsheetIdGoogleSheet   = "1UJ4pg82Sqg5F-9QRhSYEP46NJElOBmRi2mL_X_nKW84"
	dateRegexString            = `\d{2}\.\d{2}\.\d{2,4}`
	groupNameRegexString       = `[А-ЯЁ]{2,3}-\d{2,3}`
	teacherRegexString         = `([А-ЯЁ][а-яё]+)\s+([А-ЯЁ])\.`
	cabinetsRegexString        = `(\d+),(\d+)/(\d+)`
	cabinetRegexString         = `\d{3}[а-я]*\/\d{1}`
	shiftRegexString           = `\((.*?)\)`
	defaultShift               = "1 смена"
)

var (
	groupNameRegex  = regexp.MustCompile(groupNameRegexString)
	shiftRegex      = regexp.MustCompile(shiftRegexString)
	teacherRegex    = regexp.MustCompile(teacherRegexString)
	cabinetsRegex   = regexp.MustCompile(cabinetsRegexString)
	cabinetRegex    = regexp.MustCompile(cabinetRegexString)
	specialCabinets = map[string]string{
		"физ-ра": "Физ-ра",
		"кр.пол": "Кр.пол",
	}
	departments = map[int32]string{
		1: "ИТ",
		2: "ЮР",
	}
)

type ScheduleScriptServiceServer struct {
	ssov1.UnimplementedScheduleScriptServiceServer
}

type ParserScheduleGoogleSheet struct {
	DepartmentSheetName string
}

func (s ScheduleScriptServiceServer) ParseScheduleGoogleSheet(ctx context.Context, reg *ssov1.ScheduleRequest) (*ssov1.SchedulesResponse, error) {
	p := &ParserScheduleGoogleSheet{
		DepartmentSheetName: departments[reg.DepartmentId],
	}
	values, err := p.GetAllValues(ctx)
	if err != nil {
		return nil, err
	}

	currentDate := utils.GetCurrentDate()
	scheduleDate := p.ParseDate(values[0][0].(string))

	if currentDate.After(scheduleDate.AsTime()) && reg.NextDate {
		return nil, fmt.Errorf("schedule_not_found")
	}

	schedules := make([]*ssov1.ScheduleReply, 0)
	rows := make([]*ssov1.ScheduleRowReply, 0)

	dateRegex := regexp.MustCompile(dateRegexString)

	for i, row := range values {
		if i == 0 {
			continue
		}

		if len(row) > 0 && dateRegex.MatchString(fmt.Sprintf("%v", row[0])) {
			schedules = append(schedules, &ssov1.ScheduleReply{
				Date: scheduleDate,
				Rows: rows,
			})
			rows = nil
			scheduleDate = p.ParseDate(row[0].(string))
		} else {
			newRow := p.ParseRow(row)
			rows = append(rows, newRow)
		}
	}

	if len(rows) > 0 {
		schedules = append(schedules, &ssov1.ScheduleReply{
			Date: scheduleDate,
			Rows: rows,
		})
	}

	resp := &ssov1.SchedulesResponse{
		Schedules: schedules,
	}

	return resp, nil
}

func (p ParserScheduleGoogleSheet) ParseDate(dateStr string) *timestamppb.Timestamp {
	dateParts := strings.Split(dateStr, ".")
	var year, month, day int

	if len(dateParts[len(dateParts)-1]) == 2 {
		yearInt, _ := strconv.Atoi(dateParts[len(dateParts)-1])
		year = 2000 + yearInt
	} else {
		year, _ = strconv.Atoi(dateParts[len(dateParts)-1])
	}
	month, _ = strconv.Atoi(dateParts[1])
	day, _ = strconv.Atoi(dateParts[0])

	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	return timestamppb.New(t)
}

func (p ParserScheduleGoogleSheet) ParseRow(row []interface{}) *ssov1.ScheduleRowReply {
	groupName := ""
	shift := ""
	columns := make([]*ssov1.ScheduleColumnReply, 0)

	for i, column := range row {
		if i == 0 {
			groupName = groupNameRegex.FindStringSubmatch(column.(string))[0]
			shiftMatches := shiftRegex.FindStringSubmatch(column.(string))
			if len(shiftMatches) > 1 {
				shift = shiftMatches[1]
			} else {
				shift = defaultShift
			}
		} else {
			teacher := ""
			teacherMatches := teacherRegex.FindStringSubmatch(column.(string))
			if len(teacherMatches) > 1 {
				teacher = teacherMatches[1] + " " + teacherMatches[2] + "."
			}

			cabinet := ""
			cabinetsMatches := cabinetsRegex.FindAllStringSubmatch(column.(string), -1)
			if len(cabinetsMatches) > 0 {
				var formattedCabinets []string
				for _, match := range cabinetsMatches {
					formattedCabinets = append(formattedCabinets, fmt.Sprintf("%s/%s, %s/%s", match[1], match[3], match[2], match[3]))
				}
				cabinet = strings.Join(formattedCabinets, ", ")
			} else {
				cabinetMatches := cabinetRegex.FindStringSubmatch(column.(string))
				if len(cabinetMatches) > 0 {
					cabinetMatch := cabinetMatches[0]
					specialCabinet, ok := specialCabinets[cabinetMatch]
					if ok {
						cabinet = specialCabinet
					} else {
						cabinet = cabinetMatch
					}
				}
			}

			newColumn := &ssov1.ScheduleColumnReply{
				Number:  int32(i),
				Teacher: teacher,
				Cabinet: cabinet,
				Exam:    false,
			}

			columns = append(columns, newColumn)
		}
	}

	return &ssov1.ScheduleRowReply{
		GroupName: groupName,
		Shift:     shift,
		Columns:   columns,
	}
}

func (p ParserScheduleGoogleSheet) GetAllValues(ctx context.Context) ([][]interface{}, error) {
	b, err := os.ReadFile(CredentialsFileGoogleSheet)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	srv, err := sheets.NewService(ctx, option.WithCredentialsJSON(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create Sheets client: %w", err)
	}

	readRange := p.DepartmentSheetName
	resp, err := srv.Spreadsheets.Values.Get(SpreadsheetIdGoogleSheet, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get data from sheet: %w", err)
	}

	if len(resp.Values) == 0 {
		return nil, fmt.Errorf("no data found")
	}

	return resp.Values, nil
}
