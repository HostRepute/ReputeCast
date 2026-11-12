from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.signup_form_custom_field_type import SignupFormCustomFieldType
from typing import cast






T = TypeVar("T", bound="SignupFormCustomField")



@_attrs_define
class SignupFormCustomField:
    """ 
        Attributes:
            key (str):
            label (str):
            options (list[str]):
            type_ (SignupFormCustomFieldType):
     """

    key: str
    label: str
    options: list[str]
    type_: SignupFormCustomFieldType





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        label = self.label

        options = self.options



        type_ = self.type_.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "label": label,
            "options": options,
            "type": type_,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        label = d.pop("label")

        options = cast(list[str], d.pop("options"))


        type_ = SignupFormCustomFieldType(d.pop("type"))




        signup_form_custom_field = cls(
            key=key,
            label=label,
            options=options,
            type_=type_,
        )

        return signup_form_custom_field

