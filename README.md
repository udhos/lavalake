# lavalake

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


-x-
