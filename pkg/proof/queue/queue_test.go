/*
 * provendb-verify
 * Copyright (C) 2019  Southbank Software Ltd.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 *
 * https://github.com/golang-collections/collections/blob/master/queue/queue_test.go
 * @Last modified by:   guiguan
 * @Last modified time: 2019-04-02T13:28:31+11:00
 */

package queue

import (
	"testing"
)

func Test(t *testing.T) {
	q := New()

	if q.Len() != 0 {
		t.Errorf("Length should be 0")
	}

	q.Enqueue(1)

	if q.Len() != 1 {
		t.Errorf("Length should be 1")
	}

	if q.Peek().(int) != 1 {
		t.Errorf("Enqueued value should be 1")
	}

	v := q.Dequeue()

	if v.(int) != 1 {
		t.Errorf("Dequeued value should be 1")
	}

	if q.Peek() != nil || q.Dequeue() != nil {
		t.Errorf("Empty queue should have no values")
	}

	q.Enqueue(1)
	q.Enqueue(2)

	if q.Len() != 2 {
		t.Errorf("Length should be 2")
	}

	if q.Peek().(int) != 1 {
		t.Errorf("First value should be 1")
	}

	q.Dequeue()

	if q.Peek().(int) != 2 {
		t.Errorf("Next value should be 2")
	}
}
