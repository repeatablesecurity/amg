
amg is an engine for executing an automation playbook written as spec found in this project.

Unit testing is am important principle in Software development. Playbooks or automation recipes should be treated no differently. It should be possible to unit test them. Since playbooks usually contains actions carried out by a bunch of internal and external modules, it becomes hard to unit test it. This project provides an easy way to do so.

The backend binary can be started with a json file which contains a list mock scenarios for all
actions that can be executed from the playbook.
Eg: Suppose a module provides a service that returns reputation of IP address. We can pass a list pre-filled mock values that will be returned if the input criteria to the action matches.
i.e. we can specify that if "10.1.1.1" is passed as an input to an action that checks IP reputation then return the reputation as "10".
This allows us to write unit tests for our playbooks and ensure that it produces repeatable results.


Building the amg binary:
  $ cd amg; go build
Running it with a mock file:
  $./amg --mockFile=<> playbookFile=<>


