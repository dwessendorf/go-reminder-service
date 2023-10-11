#!/usr/bin/env python3
import os

import aws_cdk as cdk

from cdk_app.go_reminder_service_stack import GoReminderServiceStack


app = cdk.App()

GoReminderServiceStack(app, app.node.try_get_context('SERVICE_NAME') + "ReminderStack",)

app.synth()
