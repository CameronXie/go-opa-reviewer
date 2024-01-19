package reviewer.cfn_test

import data.reviewer.cfn.allow

mock_input(cidr) = {
  "Resources": {
    "SecurityGroup": {
      "Type": "AWS::EC2::SecurityGroup",
      "Properties": {
        "VpcId": {
          "Ref": "VPC"
        },
        "SecurityGroupIngress": [
          {
            "IpProtocol": "tcp",
            "FromPort": 80,
            "ToPort": 80,
            "CidrIp": cidr
          }
        ]
      }
    }
  }
}

test_not_allow_when_ingress_cidrip_is_anywhere {
    not allow with input as mock_input("0.0.0.0/0")
}

test_allow_when_ingress_cidrip_is_not_anywhere {
    allow with input as mock_input("10.0.0.0/25")
}
