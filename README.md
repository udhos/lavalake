# lavalake

Manage security groups uniformly across multiple cloud providers.

Install
=======

    go get github.com/udhos/lavalake
    cd ~/go/src/github.com/udhos/lavalake
    ./build.sh

Examples - Openstack
====================

List security groups:

    lake list openstack

Save security group 'group1' into file 'group1':

    lake pull openstack group1 > group1

Create security group 'group2' from file 'group1':

    lake push openstack group2 < group1

Examples - AWS
==============

List security groups:

    lake list aws

Save security group 'group1' into file 'group1':

    lake pull aws group1 vpc-id > group1

Create security group 'group2' from file 'group1':

    lake push aws group2 vpc-id < group1

Examples - Azure
================

List security groups:

    lake list azure

Save security group 'group1' into file 'group1':

    lake pull azure group1 resource-group-name > group1

Create security group 'group2' from file 'group1':

    lake push azure group2 resource-group-name < group1


-x-

