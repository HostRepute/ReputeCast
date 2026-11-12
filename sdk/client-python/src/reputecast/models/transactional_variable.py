from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_variable_type import TransactionalVariableType
from ..types import UNSET, Unset






T = TypeVar("T", bound="TransactionalVariable")



@_attrs_define
class TransactionalVariable:
    """ 
        Attributes:
            name (str):  Example: reset_url.
            required (bool):
            sensitive (bool): Never stored: logs and stored content show ••••••
            type_ (TransactionalVariableType):
            default (str | Unset):
            description (str | Unset):
            sample (str | Unset):
     """

    name: str
    required: bool
    sensitive: bool
    type_: TransactionalVariableType
    default: str | Unset = UNSET
    description: str | Unset = UNSET
    sample: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        required = self.required

        sensitive = self.sensitive

        type_ = self.type_.value

        default = self.default

        description = self.description

        sample = self.sample


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "required": required,
            "sensitive": sensitive,
            "type": type_,
        })
        if default is not UNSET:
            field_dict["default"] = default
        if description is not UNSET:
            field_dict["description"] = description
        if sample is not UNSET:
            field_dict["sample"] = sample

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        required = d.pop("required")

        sensitive = d.pop("sensitive")

        type_ = TransactionalVariableType(d.pop("type"))




        default = d.pop("default", UNSET)

        description = d.pop("description", UNSET)

        sample = d.pop("sample", UNSET)

        transactional_variable = cls(
            name=name,
            required=required,
            sensitive=sensitive,
            type_=type_,
            default=default,
            description=description,
            sample=sample,
        )

        return transactional_variable

