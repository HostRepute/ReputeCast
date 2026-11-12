from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="RoutingIssue")



@_attrs_define
class RoutingIssue:
    """ 
        Attributes:
            blocking (bool):
            code (str):
            message (str):
            stream (str):
            field (str | Unset):
            provider_id (str | Unset):
     """

    blocking: bool
    code: str
    message: str
    stream: str
    field: str | Unset = UNSET
    provider_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        blocking = self.blocking

        code = self.code

        message = self.message

        stream = self.stream

        field = self.field

        provider_id = self.provider_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocking": blocking,
            "code": code,
            "message": message,
            "stream": stream,
        })
        if field is not UNSET:
            field_dict["field"] = field
        if provider_id is not UNSET:
            field_dict["providerId"] = provider_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        blocking = d.pop("blocking")

        code = d.pop("code")

        message = d.pop("message")

        stream = d.pop("stream")

        field = d.pop("field", UNSET)

        provider_id = d.pop("providerId", UNSET)

        routing_issue = cls(
            blocking=blocking,
            code=code,
            message=message,
            stream=stream,
            field=field,
            provider_id=provider_id,
        )

        return routing_issue

