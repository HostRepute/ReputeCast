from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReportShareIncludes")



@_attrs_define
class ReportShareIncludes:
    """ 
        Attributes:
            cost (bool): The cost per sending provider
            email (bool): The email as sent, in the click map
            warmup (bool): The warm-up day of each sending provider
     """

    cost: bool
    email: bool
    warmup: bool





    def to_dict(self) -> dict[str, Any]:
        cost = self.cost

        email = self.email

        warmup = self.warmup


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cost": cost,
            "email": email,
            "warmup": warmup,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cost = d.pop("cost")

        email = d.pop("email")

        warmup = d.pop("warmup")

        report_share_includes = cls(
            cost=cost,
            email=email,
            warmup=warmup,
        )

        return report_share_includes

