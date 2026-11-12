from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AlertDomain")



@_attrs_define
class AlertDomain:
    """ 
        Attributes:
            count (int):
            domain (str):
            typo_of (str | Unset): The common domain this one looks like a typo of
     """

    count: int
    domain: str
    typo_of: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        domain = self.domain

        typo_of = self.typo_of


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "domain": domain,
        })
        if typo_of is not UNSET:
            field_dict["typoOf"] = typo_of

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        domain = d.pop("domain")

        typo_of = d.pop("typoOf", UNSET)

        alert_domain = cls(
            count=count,
            domain=domain,
            typo_of=typo_of,
        )

        return alert_domain

