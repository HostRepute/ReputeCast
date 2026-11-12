from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="ProviderCallError")



@_attrs_define
class ProviderCallError:
    """ 
        Attributes:
            at (datetime.datetime):
            code (str):
            message (str):
            op (str):
            status (int):
     """

    at: datetime.datetime
    code: str
    message: str
    op: str
    status: int





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        code = self.code

        message = self.message

        op = self.op

        status = self.status


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "code": code,
            "message": message,
            "op": op,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        code = d.pop("code")

        message = d.pop("message")

        op = d.pop("op")

        status = d.pop("status")

        provider_call_error = cls(
            at=at,
            code=code,
            message=message,
            op=op,
            status=status,
        )

        return provider_call_error

