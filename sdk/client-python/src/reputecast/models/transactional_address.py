from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="TransactionalAddress")



@_attrs_define
class TransactionalAddress:
    """ 
        Attributes:
            email (str):  Example: sam.rivera@outlook.com.
            name (str | Unset):  Example: Sam Rivera.
     """

    email: str
    name: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        email = self.email

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
        })
        if name is not UNSET:
            field_dict["name"] = name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        email = d.pop("email")

        name = d.pop("name", UNSET)

        transactional_address = cls(
            email=email,
            name=name,
        )

        return transactional_address

