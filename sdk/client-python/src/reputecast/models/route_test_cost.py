from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.route_test_cost_basis import RouteTestCostBasis
from ..types import UNSET, Unset






T = TypeVar("T", bound="RouteTestCost")



@_attrs_define
class RouteTestCost:
    """ 
        Attributes:
            basis (RouteTestCostBasis):
            per1000 (float | Unset):
            per_message (float | Unset):
     """

    basis: RouteTestCostBasis
    per1000: float | Unset = UNSET
    per_message: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        basis = self.basis.value

        per1000 = self.per1000

        per_message = self.per_message


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "basis": basis,
        })
        if per1000 is not UNSET:
            field_dict["per1000"] = per1000
        if per_message is not UNSET:
            field_dict["perMessage"] = per_message

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        basis = RouteTestCostBasis(d.pop("basis"))




        per1000 = d.pop("per1000", UNSET)

        per_message = d.pop("perMessage", UNSET)

        route_test_cost = cls(
            basis=basis,
            per1000=per1000,
            per_message=per_message,
        )

        return route_test_cost

