class APIException(Exception):
    def __init__(
        self,
        msg=None,
        url=None,
        method=None,
        body=None,
        status_code=None,
        response_text=None,
    ):
        self.msg = msg
        self.url = url
        self.method = method
        self.body = body
        self.status_code = status_code
        self.response_text = response_text

        super().__init__(msg)

    def __str__(self):
        return f"""{self.msg} on {self.url} with method {self.method}:
Body:
{self.body}

Response:
@StatusCode: {self.status_code}
@ResponseText: {self.response_text}
        """
