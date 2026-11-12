from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.settings_provider_cost_basis import SettingsProviderCostBasis






T = TypeVar("T", bound="SettingsProviderCost")



@_attrs_define
class SettingsProviderCost:
    """ 
        Attributes:
            basis (SettingsProviderCostBasis):
            monthly (float):
            to_date (float):
     """

    basis: SettingsProviderCostBasis
    monthly: float
    to_date: float





    def to_dict(self) -> dict[str, Any]:
        basis = self.basis.value

        monthly = self.monthly

        to_date = self.to_date


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "basis": basis,
            "monthly": monthly,
            "toDate": to_date,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        basis = SettingsProviderCostBasis(d.pop("basis"))




        monthly = d.pop("monthly")

        to_date = d.pop("toDate")

        settings_provider_cost = cls(
            basis=basis,
            monthly=monthly,
            to_date=to_date,
        )

        return settings_provider_cost

