from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SignupFormCopyField")



@_attrs_define
class SignupFormCopyField:
    """ 
        Attributes:
            key (str):
            source (str): Its label in the form's own language
     """

    key: str
    source: str





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        source = self.source


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "source": source,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        source = d.pop("source")

        signup_form_copy_field = cls(
            key=key,
            source=source,
        )

        return signup_form_copy_field

