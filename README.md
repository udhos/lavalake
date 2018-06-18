[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/lavalake/blob/master/LICENSE)
[![Go Report Card - lavalake](https://goreportcard.com/badge/github.com/udhos/lavalake)](https://goreportcard.com/report/github.com/udhos/lavalake)

# lavalake

Manage security groups uniformly across multiple cloud providers.

- Fetch a security group from a cloud provider and save it as YAML file.
- Can edit the file locally then send it back to the cloud provider.
- Can write the security group rules to cloud provider as another group.
- Can send the security group to another cloud provider, unless the group rules use features unsupported by target provider.

Install
=======

    go get github.com/udhos/lavalake
    cd ~/go/src/github.com/udhos/lavalake
    ./build.sh

Examples - Openstack
====================

List security groups:

    lake list openstack

Save security group 'group1' into file 'group1.yaml':

    lake pull openstack group1 > group1.yaml

Create/update security group 'group2' from file 'group1.yaml':

    lake push openstack group2 < group1.yaml

Examples - AWS
==============

List security groups:

    lake list aws [vpc-id]

Save security group 'group1' into file 'group1.yaml':

    lake pull aws group1 vpc-id > group1.yaml

Create/update security group 'group2' from file 'group1.yaml':

    lake push aws group2 vpc-id < group1.yaml

Examples - Azure
================

List security groups:

    lake list azure

Save security group 'group1' into file 'group1.yaml':

    lake pull azure group1 resource-group-name > group1.yaml

Create/update security group 'group2' from file 'group1.yaml':

    lake push azure group2 resource-group-name location < group1.yaml


-x-

