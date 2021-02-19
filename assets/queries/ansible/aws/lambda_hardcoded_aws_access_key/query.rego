package Cx
import data.generic.ansible as ansLib

CxPolicy[result] {
	document = input.document[i]
	tasks := ansLib.getTasks(document)
	awsLambda := tasks[_]
	awsLambdaBody := awsLambda["community.aws.lambda"]

	awsLambdaName := awsLambda.name
	regex.match(`^[A-Za-z0-9/+=]{40}$`, awsLambdaBody.aws_access_key)
	result := {
		"documentId": input.document[i].id,
		"searchKey": sprintf("name={{%s}}.aws_access_key", [awsLambdaName]),
		"issueType": "IncorrectValue",
		"keyExpectedValue": sprintf("{{%s}}.aws_access_key should not be in plaintext.", [awsLambdaName]),
		"keyActualValue": sprintf("{{%s}}aws_access_key is in plaintext ", [awsLambdaName, awsLambdaBody.name]),
	}
}