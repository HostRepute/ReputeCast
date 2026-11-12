from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VisibilityCatalogScore")



@_attrs_define
class VisibilityCatalogScore:
    """ 
        Attributes:
            configured (bool):
            hot_from (int):
            warm_from (int):
     """

    configured: bool
    hot_from: int
    warm_from: int





    def to_dict(self) -> dict[str, Any]:
        configured = self.configured

        hot_from = self.hot_from

        warm_from = self.warm_from


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "configured": configured,
            "hotFrom": hot_from,
            "warmFrom": warm_from,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        configured = d.pop("configured")

        hot_from = d.pop("hotFrom")

        warm_from = d.pop("warmFrom")

        visibility_catalog_score = cls(
            configured=configured,
            hot_from=hot_from,
            warm_from=warm_from,
        )

        return visibility_catalog_score

