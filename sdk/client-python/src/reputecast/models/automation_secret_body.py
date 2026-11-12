from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AutomationSecretBody")



@_attrs_define
class AutomationSecretBody:
    """ 
        Attributes:
            secret (str):
     """

    secret: str





    def to_dict(self) -> dict[str, Any]:
        secret = self.secret


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "secret": secret,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        secret = d.pop("secret")

        automation_secret_body = cls(
            secret=secret,
        )

        return automation_secret_body

