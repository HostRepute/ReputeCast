from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="AuditCategory")



@_attrs_define
class AuditCategory:
    """ 
        Attributes:
            groups (list[str]):
            key (str):
            label (str):
     """

    groups: list[str]
    key: str
    label: str





    def to_dict(self) -> dict[str, Any]:
        groups = self.groups



        key = self.key

        label = self.label


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "groups": groups,
            "key": key,
            "label": label,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        groups = cast(list[str], d.pop("groups"))


        key = d.pop("key")

        label = d.pop("label")

        audit_category = cls(
            groups=groups,
            key=key,
            label=label,
        )

        return audit_category

