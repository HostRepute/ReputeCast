from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="IntegrationResolution")



@_attrs_define
class IntegrationResolution:
    """ 
        Attributes:
            later_runs (list[datetime.datetime]):
            fixed_at (datetime.datetime | Unset):
            fixed_by (str | Unset):
     """

    later_runs: list[datetime.datetime]
    fixed_at: datetime.datetime | Unset = UNSET
    fixed_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        later_runs = []
        for later_runs_item_data in self.later_runs:
            later_runs_item = later_runs_item_data.isoformat()
            later_runs.append(later_runs_item)



        fixed_at: str | Unset = UNSET
        if not isinstance(self.fixed_at, Unset):
            fixed_at = self.fixed_at.isoformat()

        fixed_by = self.fixed_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "laterRuns": later_runs,
        })
        if fixed_at is not UNSET:
            field_dict["fixedAt"] = fixed_at
        if fixed_by is not UNSET:
            field_dict["fixedBy"] = fixed_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        later_runs = []
        _later_runs = d.pop("laterRuns")
        for later_runs_item_data in (_later_runs):
            later_runs_item = datetime.datetime.fromisoformat(later_runs_item_data)



            later_runs.append(later_runs_item)


        _fixed_at = d.pop("fixedAt", UNSET)
        fixed_at: datetime.datetime | Unset
        if isinstance(_fixed_at,  Unset):
            fixed_at = UNSET
        else:
            fixed_at = datetime.datetime.fromisoformat(_fixed_at)




        fixed_by = d.pop("fixedBy", UNSET)

        integration_resolution = cls(
            later_runs=later_runs,
            fixed_at=fixed_at,
            fixed_by=fixed_by,
        )

        return integration_resolution

