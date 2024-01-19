package reviewer.cfn

import rego.v1

default allow := false

allow if {
    count(violation) == 0
}

violation contains id if {
    some id
    input.Resources[id].Type == "AWS::EC2::SecurityGroup"
    input.Resources[id].Properties.SecurityGroupIngress[_].CidrIp == "0.0.0.0/0"
}
