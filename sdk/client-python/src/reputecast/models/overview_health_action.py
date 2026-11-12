from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.overview_health_action_kind import OverviewHealthActionKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="OverviewHealthAction")



@_attrs_define
class OverviewHealthAction:
    """ 
        Attributes:
            kind (OverviewHealthActionKind):
            label (str): request delisting
            asset (str | Unset):
     """

    kind: OverviewHealthActionKind
    label: str
    asset: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        label = self.label

        asset = self.asset


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "label": label,
        })
        if asset is not UNSET:
            field_dict["asset"] = asset

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = OverviewHealthActionKind(d.pop("kind"))




        label = d.pop("label")

        asset = d.pop("asset", UNSET)

        overview_health_action = cls(
            kind=kind,
            label=label,
            asset=asset,
        )

        return overview_health_action

