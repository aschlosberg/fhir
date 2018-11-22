/*
 * Copyright 2018 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Helpers for working with FHIRMessages in native Go data types.

package stu3

import "time"

// Time returns the Date object as a UTC Time, rounded based on Precision.
// Rounding is not performed with time.Time.Round() but rather by setting all
// finer-grain time scales to "zero";" i.e. YEAR-precision will always be
// January 1st at 00:00, MONTH-precision will always be 1st of the month at
// 00:00, and DAY will always be at 00:00 time.
func (p *Date) Time() time.Time {
	t := utcFromMicroSec(p.ValueUs)

	// 00:00 time
	sub := time.Duration(t.Hour())*time.Hour +
		time.Duration(t.Minute())*time.Minute +
		time.Duration(t.Second())*time.Second
	t = t.Add(-sub)

	var subMonths, subDays int
	switch p.Precision {
	case Date_YEAR:
		subMonths = int(t.Month() - time.January)
		fallthrough
	case Date_MONTH:
		subDays = t.Day() - 1
	}

	return t.AddDate(0, -subMonths, -subDays)
}

// TODO(arrans): implement AsDate()
// AsDate returns a Date at the specified time, after rounding according to the
// specified precision.
//func AsDate(t time.Time, p Date_Precision) *Date {
//}

// TODO(arrans) implement Decimal proto to numeric equivalent helpers. Should
// these use math/big.Float?
