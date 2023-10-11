from aws_cdk import (
    Duration,
    Stack,
    aws_lambda_go_alpha as go,
    aws_iam as _iam,
    aws_events as events,
    aws_events_targets as targets
)
from constructs import Construct

class GoReminderServiceStack(Stack):

    def __init__(self, scope: Construct, construct_id: str, **kwargs) -> None:
        super().__init__(scope, construct_id, **kwargs)

        # Define your base policy
        lambda_policy = _iam.ManagedPolicy(self, id= self.node.try_get_context('SERVICE_NAME'
                                                                               '') + "LambdaPolicy",
                                                statements=[
                                                    _iam.PolicyStatement(
                                                        actions=[
                                                            "logs:CreateLogGroup",
                                                            "logs:CreateLogStream",
                                                            "logs:PutLogEvents"
                                                        ],
                                                        resources=["arn:aws:logs:*:*:*"]
                                                    ),
                                                    _iam.PolicyStatement(
                                                        actions=[
                                                            "secretsmanager:DescribeSecret",
                                                            "secretsmanager:GetSecretValue"
                                                        ],
                                                        resources=["*"]
                                                    )
                                                ]
                                                )

        # Define the role for the Lambda Function
        lambda_role = _iam.Role(scope=self, id = self.node.try_get_context("SERVICE_NAME") + "LambdaRole",
                                    assumed_by =_iam.ServicePrincipal('lambda.amazonaws.com'),
                                    managed_policies=[lambda_policy])

        goLambda_evening = go.GoFunction(self, self.node.try_get_context("SERVICE_NAME") + "ReminderEvening",
            entry="lambda/go-reminder-service",
            function_name=self.node.try_get_context("SERVICE_NAME") + "ReminderEvening",
            role=lambda_role,
            timeout=Duration.seconds(30),
            environment={
                    "MODE": "EVENING",
                    "GOOGLE_CREDENTIALS_SECRET_NAME": self.node.try_get_context("GOOGLE_CREDENTIALS_SECRET_NAME"),
                    "GOOGLE_SMTP_CREDENTIALS_SECRET_NAME": self.node.try_get_context("GOOGLE_SMTP_CREDENTIALS_SECRET_NAME"),
                    "GOOGLE_SHEETS_ID_PLAN": self.node.try_get_context("GOOGLE_SHEETS_ID_PLAN"),
                    "GOOGLE_SHEETS_RANGE_PLAN": self.node.try_get_context("GOOGLE_SHEETS_RANGE_PLAN"),
                    "GOOGLE_SHEETS_RANGE_MAPPING": self.node.try_get_context("GOOGLE_SHEETS_RANGE_MAPPING"),
                    "GOOGLE_SHEETS_ID_ADDRESSES": self.node.try_get_context("GOOGLE_SHEETS_ID_ADDRESSES"),
                    "GOOGLE_SHEETS_RANGE_ADDRESSES": self.node.try_get_context("GOOGLE_SHEETS_RANGE_ADDRESSES"),
                    "GOOGLE_SHEETS_ID_OPTIONS": self.node.try_get_context("GOOGLE_SHEETS_ID_OPTIONS"),
                    "GOOGLE_SHEETS_RANGE_OPTIONS": self.node.try_get_context("GOOGLE_SHEETS_RANGE_OPTIONS")
            }
        )

        goLambda_morning = go.GoFunction(self, self.node.try_get_context("SERVICE_NAME") + "ReminderMorning",
            entry="lambda/go-reminder-service",
            function_name=self.node.try_get_context("SERVICE_NAME") + "ReminderMorning",
            role=lambda_role,
            timeout=Duration.seconds(30),
            environment={
                    "MODE": "MORNING",
                    "GOOGLE_CREDENTIALS_SECRET_NAME": self.node.try_get_context("GOOGLE_CREDENTIALS_SECRET_NAME"),
                    "GOOGLE_SMTP_CREDENTIALS_SECRET_NAME": self.node.try_get_context("GOOGLE_SMTP_CREDENTIALS_SECRET_NAME"),
                    "GOOGLE_SHEETS_ID_PLAN": self.node.try_get_context("GOOGLE_SHEETS_ID_PLAN"),
                    "GOOGLE_SHEETS_RANGE_PLAN": self.node.try_get_context("GOOGLE_SHEETS_RANGE_PLAN"),
                    "GOOGLE_SHEETS_RANGE_MAPPING": self.node.try_get_context("GOOGLE_SHEETS_RANGE_MAPPING"),
                    "GOOGLE_SHEETS_ID_ADDRESSES": self.node.try_get_context("GOOGLE_SHEETS_ID_ADDRESSES"),
                    "GOOGLE_SHEETS_RANGE_ADDRESSES": self.node.try_get_context("GOOGLE_SHEETS_RANGE_ADDRESSES"),
                    "GOOGLE_SHEETS_ID_OPTIONS": self.node.try_get_context("GOOGLE_SHEETS_ID_OPTIONS"),
                    "GOOGLE_SHEETS_RANGE_OPTIONS": self.node.try_get_context("GOOGLE_SHEETS_RANGE_OPTIONS")
            }
        )

        # Create an EventBridge rule for the morning
        rule_morning = events.Rule(
            self, 'TrafficLightsServiceEventbridgeRuleMorning',
            rule_name=self.node.try_get_context("SERVICE_NAME") + "Morning",
            schedule=events.Schedule.cron(
                minute=self.node.try_get_context("MORNING_CRON_MINUTE"),
                hour=self.node.try_get_context("MORNING_CRON_HOUR"),
                month=self.node.try_get_context("MORNING_CRON_MONTH"),
                week_day=self.node.try_get_context("MORNING_CRON_WEEKDAY"),
                year=self.node.try_get_context("MORNING_CRON_YEAR"),
            )
        )
        # Add the Lambda function as a target to the rule
        rule_morning.add_target(targets.LambdaFunction(goLambda_morning))

        # Create an EventBridge rule for the evening
        rule_evening = events.Rule(
            self, 'TrafficLightsServiceEventbridgeRuleEvening',
            rule_name=self.node.try_get_context("SERVICE_NAME") + "Evening",
            schedule=events.Schedule.cron(
                minute=self.node.try_get_context("EVENING_CRON_MINUTE"),
                hour=self.node.try_get_context("EVENING_CRON_HOUR"),
                month=self.node.try_get_context("EVENING_CRON_MONTH"),
                week_day=self.node.try_get_context("EVENING_CRON_WEEKDAY"),
                year=self.node.try_get_context("EVENING_CRON_YEAR"),
            )
        )

        # Add the Lambda function as a target to the rule
        rule_evening.add_target(targets.LambdaFunction(goLambda_evening))

        

