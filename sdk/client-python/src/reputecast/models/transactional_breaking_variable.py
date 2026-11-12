from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="TransactionalBreakingVariable")



@_attrs_define
class TransactionalBreakingVariable:
    """ 
        Attributes:
            calls_without (int):
            name (str):
            suggested_default (str):
            type_ (str):
     """

    calls_without: int
    name: str
    suggested_default: str
    type_: str





    def to_dict(self) -> dict[str, Any]:
        calls_without = self.calls_without

        name = self.name

        suggested_default = self.suggested_default

        type_ = self.type_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "callsWithout": calls_without,
            "name": name,
            "suggestedDefault": suggested_default,
            "type": type_,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        calls_without = d.pop("callsWithout")

        name = d.pop("name")

        suggested_default = d.pop("suggestedDefault")

        type_ = d.pop("type")

        transactional_breaking_variable = cls(
            calls_without=calls_without,
            name=name,
            suggested_default=suggested_default,
            type_=type_,
        )

        return transactional_breaking_variable

