from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.revenue_mark_kind import RevenueMarkKind
from typing import cast
import datetime






T = TypeVar("T", bound="RevenueMark")



@_attrs_define
class RevenueMark:
    """ 
        Attributes:
            at (datetime.datetime):
            kind (RevenueMarkKind):
            n (int):
            name (str):
     """

    at: datetime.datetime
    kind: RevenueMarkKind
    n: int
    name: str





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        kind = self.kind.value

        n = self.n

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "kind": kind,
            "n": n,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        kind = RevenueMarkKind(d.pop("kind"))




        n = d.pop("n")

        name = d.pop("name")

        revenue_mark = cls(
            at=at,
            kind=kind,
            n=n,
            name=name,
        )

        return revenue_mark

