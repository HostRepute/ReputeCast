from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SignupFormField")



@_attrs_define
class SignupFormField:
    """ 
        Attributes:
            key (str): email, first_name, last_name, company or a custom field key
            label (str): Shown above the input; empty uses the field's name
            required (bool):
     """

    key: str
    label: str
    required: bool





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        label = self.label

        required = self.required


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "label": label,
            "required": required,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        label = d.pop("label")

        required = d.pop("required")

        signup_form_field = cls(
            key=key,
            label=label,
            required=required,
        )

        return signup_form_field

