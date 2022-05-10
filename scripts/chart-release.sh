#!/usr/bin/env bash

# Copyright paskal.maksim@gmail.com
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
set -ex

owner=maksim-paskal
repo=pod-admission-controller

export CR_RELEASE_NAME_TEMPLATE="helm-chart-{{ .Version }}"

rm -rf .cr-*
mkdir -p .cr-index
cr package ./charts/pod-admission-controller
cr upload -o $owner -r $repo -c "$(git rev-parse HEAD)"
cr index -o $owner -r $repo -c https://$owner.github.io/$repo