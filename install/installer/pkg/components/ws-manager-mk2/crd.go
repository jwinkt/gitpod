// Copyright (c) 2022 Gitpod GmbH. All rights reserved.
// Licensed under the MIT License. See License-MIT.txt in the project root for license information.

package wsmanagermk2

import (
	_ "embed"

	"github.com/gitpod-io/gitpod/installer/pkg/common"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

// todo(sje): establish how to pass in config with cw
func crd(ctx *common.RenderContext) ([]runtime.Object, error) {
	var res apiextensions.CustomResourceDefinition
	err := yaml.Unmarshal([]byte(crdYAML), &res)
	if err != nil {
		return nil, err
	}

	return []runtime.Object{
		&res,
	}, nil
}

//go:embed crd.yaml
var crdYAML string
