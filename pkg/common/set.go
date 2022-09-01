/*
 Copyright 2022 NTT Communications Corporation.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package common

type Empty struct{}

type Set[T comparable] struct {
	m map[T]Empty
}

func NewSet[T comparable](items ...T) Set[T] {
	s := Set[T]{
		m: make(map[T]Empty),
	}
	for _, i := range items {
		s.Add(i)
	}
	return s
}

func (s *Set[T]) Add(v T) bool {
	added := !s.Has(v)
	s.m[v] = Empty{}
	return added
}

func (s *Set[T]) Has(v T) bool {
	_, ok := s.m[v]
	return ok
}

func (s *Set[T]) List() []T {
	var items []T
	for k, _ := range s.m {
		items = append(items, k)
	}
	return items
}
