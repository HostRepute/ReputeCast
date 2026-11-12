from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="UpgradeFAQ")



@_attrs_define
class UpgradeFAQ:
    """ 
        Attributes:
            a (str):
            q (str):
     """

    a: str
    q: str





    def to_dict(self) -> dict[str, Any]:
        a = self.a

        q = self.q


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "a": a,
            "q": q,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        a = d.pop("a")

        q = d.pop("q")

        upgrade_faq = cls(
            a=a,
            q=q,
        )

        return upgrade_faq

