from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.growth_retention import GrowthRetention





T = TypeVar("T", bound="GrowthCohort")



@_attrs_define
class GrowthCohort:
    """ 
        Attributes:
            after30 (GrowthRetention):
            after90 (GrowthRetention):
            contacts (int):
            month (str): YYYY-MM
     """

    after30: GrowthRetention
    after90: GrowthRetention
    contacts: int
    month: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.growth_retention import GrowthRetention # noqa: PLC0415
        after30 = self.after30.to_dict()

        after90 = self.after90.to_dict()

        contacts = self.contacts

        month = self.month


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "after30": after30,
            "after90": after90,
            "contacts": contacts,
            "month": month,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.growth_retention import GrowthRetention # noqa: PLC0415
        d = dict(src_dict)
        after30 = GrowthRetention.from_dict(d.pop("after30"))




        after90 = GrowthRetention.from_dict(d.pop("after90"))




        contacts = d.pop("contacts")

        month = d.pop("month")

        growth_cohort = cls(
            after30=after30,
            after90=after90,
            contacts=contacts,
            month=month,
        )

        return growth_cohort

