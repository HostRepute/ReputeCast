from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="DigestToken")



@_attrs_define
class DigestToken:
    """ 
        Attributes:
            token (str):
            value (str):
     """

    token: str
    value: str





    def to_dict(self) -> dict[str, Any]:
        token = self.token

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "token": token,
            "value": value,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        token = d.pop("token")

        value = d.pop("value")

        digest_token = cls(
            token=token,
            value=value,
        )

        return digest_token

