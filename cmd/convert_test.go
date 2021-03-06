/*
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2016 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package cmd

import (
	"bytes"
	"testing"

	"io/ioutil"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

const testHAR = `
{
	"log": {
		"version": "1.2",
		"creator": {
		"name": "WebInspector",
		"version": "537.36"
		},
		"pages": [
		{
			"startedDateTime": "2018-01-21T19:48:40.432Z",
			"id": "page_2",
			"title": "https://golang.org/",
			"pageTimings": {
			"onContentLoad": 590.3389999875799,
			"onLoad": 1593.1009999476373
			}
		}
		],
		"entries": [
		{
			"startedDateTime": "2018-01-21T19:48:40.587Z",
			"time": 147.5899999756366,
			"request": {
				"method": "GET",
				"url": "https://golang.org/",
				"httpVersion": "http/2.0+quic/39",
				"headers": [
					{
					"name": "pragma",
					"value": "no-cache"
					}
				],
				"queryString": [],
				"cookies": [],
				"headersSize": -1,
				"bodySize": 0
			},
			"cache": {},
			"timings": {
				"blocked": 0.43399997614324004,
				"dns": -1,
				"ssl": -1,
				"connect": -1,
				"send": 0.12700003571808005,
				"wait": 149.02899996377528,
				"receive": 0,
				"_blocked_queueing": -1
			},
			"serverIPAddress": "172.217.22.177",
			"pageref": "page_2"
		}
		]
	}
}
`

const testHARConvertResult = `import { group, sleep } from 'k6';
import http from 'k6/http';

// Version: 1.2
// Creator: WebInspector

export let options = { maxRedirects: 0 };

export default function() {

	group("page_2 - https://golang.org/", function() {
		let req, res;
		req = [{
			"method": "get",
			"url": "https://golang.org/",
			"params": {
				"headers": {
					"pragma": "no-cache"
				}
			}
		}];
		res = http.batch(req);
		// Random sleep between 20s and 40s
		sleep(Math.floor(Math.random()*20+20));
	});

}
`

func TestIntegrationConvertCmd(t *testing.T) {
	t.Run("Correlate", func(t *testing.T) {
		har, err := ioutil.ReadFile("testdata/example.har")
		assert.NoError(t, err)

		expectedTestPlan, err := ioutil.ReadFile("testdata/example.js")
		assert.NoError(t, err)

		defaultFs = afero.NewMemMapFs()
		err = afero.WriteFile(defaultFs, "/input.har", []byte(har), 0644)
		assert.NoError(t, err)

		buf := &bytes.Buffer{}
		defaultWriter = buf

		convertCmd.Flags().Set("correlate", "true")
		convertCmd.Flags().Set("no-batch", "true")
		convertCmd.Flags().Set("enable-status-code-checks", "true")
		convertCmd.Flags().Set("return-on-failed-check", "true")

		err = convertCmd.RunE(convertCmd, []string{"/input.har"})

		// reset the convertCmd to default flags. There must be a nicer and less error prone way to do this...
		convertCmd.Flags().Set("correlate", "false")
		convertCmd.Flags().Set("no-batch", "false")
		convertCmd.Flags().Set("enable-status-code-checks", "false")
		convertCmd.Flags().Set("return-on-failed-check", "false")

		if assert.NoError(t, err) {
			// assert.Equal suppresses the diff it is too big, so we add it as the test error message manually as well.
			diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
				A:        difflib.SplitLines(string(expectedTestPlan)),
				B:        difflib.SplitLines(buf.String()),
				FromFile: "Expected",
				FromDate: "",
				ToFile:   "Actual",
				ToDate:   "",
				Context:  1,
			})
			assert.Equal(t, string(expectedTestPlan), buf.String(), diff)
		}
	})
	t.Run("Stdout", func(t *testing.T) {
		defaultFs = afero.NewMemMapFs()
		err := afero.WriteFile(defaultFs, "/input.har", []byte(testHAR), 0644)
		assert.NoError(t, err)

		buf := &bytes.Buffer{}
		defaultWriter = buf

		err = convertCmd.RunE(convertCmd, []string{"/input.har"})
		assert.NoError(t, err)
		assert.Equal(t, testHARConvertResult, buf.String())
	})
	t.Run("Output file", func(t *testing.T) {
		defaultFs = afero.NewMemMapFs()
		err := afero.WriteFile(defaultFs, "/input.har", []byte(testHAR), 0644)
		assert.NoError(t, err)

		err = convertCmd.Flags().Set("output", "/output.js")
		assert.NoError(t, err)
		err = convertCmd.RunE(convertCmd, []string{"/input.har"})
		assert.NoError(t, err)

		output, err := afero.ReadFile(defaultFs, "/output.js")
		assert.NoError(t, err)
		assert.Equal(t, testHARConvertResult, string(output))
	})
}
