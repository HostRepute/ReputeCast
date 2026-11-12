from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_cost_to_date_basis import ProviderCostToDateBasis
from ..types import UNSET, Unset






T = TypeVar("T", bound="ProviderCostToDate")



@_attrs_define
class ProviderCostToDate:
    """ 
        Attributes:
            basis (ProviderCostToDateBasis):
            to_date (float): Dollars this month
            monthly (float | Unset):
            per1000 (float | Unset):
            source (str | Unset):
     """

    basis: ProviderCostToDateBasis
    to_date: float
    monthly: float | Unset = UNSET
    per1000: float | Unset = UNSET
    source: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        basis = self.basis.value

        to_date = self.to_date

        monthly = self.monthly

        per1000 = self.per1000

        source = self.source


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "basis": basis,
            "toDate": to_date,
        })
        if monthly is not UNSET:
            field_dict["monthly"] = monthly
        if per1000 is not UNSET:
            field_dict["per1000"] = per1000
        if source is not UNSET:
            field_dict["source"] = source

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        basis = ProviderCostToDateBasis(d.pop("basis"))




        to_date = d.pop("toDate")

        monthly = d.pop("monthly", UNSET)

        per1000 = d.pop("per1000", UNSET)

        source = d.pop("source", UNSET)

        provider_cost_to_date = cls(
            basis=basis,
            to_date=to_date,
            monthly=monthly,
            per1000=per1000,
            source=source,
        )

        return provider_cost_to_date

