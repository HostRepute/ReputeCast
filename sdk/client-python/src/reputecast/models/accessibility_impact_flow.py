from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AccessibilityImpactFlow")



@_attrs_define
class AccessibilityImpactFlow:
    """ 
        Attributes:
            automation_id (str):
            name (str):
            serious (int):
            step (str):
            summary (str):
            version (int):
            live_version (int | Unset):
     """

    automation_id: str
    name: str
    serious: int
    step: str
    summary: str
    version: int
    live_version: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        automation_id = self.automation_id

        name = self.name

        serious = self.serious

        step = self.step

        summary = self.summary

        version = self.version

        live_version = self.live_version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "automationId": automation_id,
            "name": name,
            "serious": serious,
            "step": step,
            "summary": summary,
            "version": version,
        })
        if live_version is not UNSET:
            field_dict["liveVersion"] = live_version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        automation_id = d.pop("automationId")

        name = d.pop("name")

        serious = d.pop("serious")

        step = d.pop("step")

        summary = d.pop("summary")

        version = d.pop("version")

        live_version = d.pop("liveVersion", UNSET)

        accessibility_impact_flow = cls(
            automation_id=automation_id,
            name=name,
            serious=serious,
            step=step,
            summary=summary,
            version=version,
            live_version=live_version,
        )

        return accessibility_impact_flow

