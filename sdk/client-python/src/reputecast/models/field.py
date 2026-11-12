from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="Field")



@_attrs_define
class Field:
    """ 
        Attributes:
            key (str):
            label (str):
            required (bool):
            secret (bool):
            default (str | Unset):
            help_ (str | Unset):
            options (list[str] | Unset):
     """

    key: str
    label: str
    required: bool
    secret: bool
    default: str | Unset = UNSET
    help_: str | Unset = UNSET
    options: list[str] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        label = self.label

        required = self.required

        secret = self.secret

        default = self.default

        help_ = self.help_

        options: list[str] | Unset = UNSET
        if not isinstance(self.options, Unset):
            options = self.options




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "label": label,
            "required": required,
            "secret": secret,
        })
        if default is not UNSET:
            field_dict["default"] = default
        if help_ is not UNSET:
            field_dict["help"] = help_
        if options is not UNSET:
            field_dict["options"] = options

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        label = d.pop("label")

        required = d.pop("required")

        secret = d.pop("secret")

        default = d.pop("default", UNSET)

        help_ = d.pop("help", UNSET)

        options = cast(list[str], d.pop("options", UNSET))


        field = cls(
            key=key,
            label=label,
            required=required,
            secret=secret,
            default=default,
            help_=help_,
            options=options,
        )

        return field

