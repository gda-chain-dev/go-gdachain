Pod::Spec.new do |spec|
  spec.name         = 'Ggda'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/gdachain/go-gdachain'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS gdachain Client'
  spec.source       = { :git => 'https://github.com/gdachain/go-gdachain.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Ggda.framework'

	spec.prepare_command = <<-CMD
    curl https://ggdastore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Ggda.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end
