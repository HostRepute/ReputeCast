from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ConnectError")



@_attrs_define
class ConnectError:
    """ 
        Attributes:
            message (str):
            code (str | Unset):
            detail (str | Unset):
            field (str | Unset):
     """

    message: str
    code: str | Unset = UNSET
    detail: str | Unset = UNSET
    field: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        message = self.message

        code = self.code

        detail = self.detail

        field = self.field


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "message": message,
        })
        if code is not UNSET:
            field_dict["code"] = code
        if detail is not UNSET:
            field_dict["detail"] = detail
        if field is not UNSET:
            field_dict["field"] = field

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        message = d.pop("message")

        code = d.pop("code", UNSET)

        detail = d.pop("detail", UNSET)

        field = d.pop("field", UNSET)

        connect_error = cls(
            message=message,
            code=code,
            detail=detail,
            field=field,
        )

        return connect_error

