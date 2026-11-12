from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.check_line_status import CheckLineStatus






T = TypeVar("T", bound="CheckLine")



@_attrs_define
class CheckLine:
    """ 
        Attributes:
            detail (str):
            status (CheckLineStatus):
     """

    detail: str
    status: CheckLineStatus





    def to_dict(self) -> dict[str, Any]:
        detail = self.detail

        status = self.status.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "detail": detail,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        detail = d.pop("detail")

        status = CheckLineStatus(d.pop("status"))




        check_line = cls(
            detail=detail,
            status=status,
        )

        return check_line

