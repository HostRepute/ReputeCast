from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="TransactionalErrorCode")



@_attrs_define
class TransactionalErrorCode:
    """ 
        Attributes:
            code (str):
            status (int):
     """

    code: str
    status: int





    def to_dict(self) -> dict[str, Any]:
        code = self.code

        status = self.status


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "code": code,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        code = d.pop("code")

        status = d.pop("status")

        transactional_error_code = cls(
            code=code,
            status=status,
        )

        return transactional_error_code

