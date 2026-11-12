from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VerificationReplaceKeyInput")



@_attrs_define
class VerificationReplaceKeyInput:
    """ 
        Attributes:
            api_key (str):
     """

    api_key: str





    def to_dict(self) -> dict[str, Any]:
        api_key = self.api_key


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "apiKey": api_key,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        api_key = d.pop("apiKey")

        verification_replace_key_input = cls(
            api_key=api_key,
        )

        return verification_replace_key_input

