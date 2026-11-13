from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.earn_price import EarnPrice





T = TypeVar("T", bound="EarnGroupPrices")



@_attrs_define
class EarnGroupPrices:
    """ 
        Attributes:
            seats (int):
            seats_max (int):
            seats_min (int):
            teams (int):
            monthly (EarnPrice | Unset):
            yearly (EarnPrice | Unset):
     """

    seats: int
    seats_max: int
    seats_min: int
    teams: int
    monthly: EarnPrice | Unset = UNSET
    yearly: EarnPrice | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_price import EarnPrice # noqa: PLC0415
        seats = self.seats

        seats_max = self.seats_max

        seats_min = self.seats_min

        teams = self.teams

        monthly: dict[str, Any] | Unset = UNSET
        if not isinstance(self.monthly, Unset):
            monthly = self.monthly.to_dict()

        yearly: dict[str, Any] | Unset = UNSET
        if not isinstance(self.yearly, Unset):
            yearly = self.yearly.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "seats": seats,
            "seatsMax": seats_max,
            "seatsMin": seats_min,
            "teams": teams,
        })
        if monthly is not UNSET:
            field_dict["monthly"] = monthly
        if yearly is not UNSET:
            field_dict["yearly"] = yearly

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_price import EarnPrice # noqa: PLC0415
        d = dict(src_dict)
        seats = d.pop("seats")

        seats_max = d.pop("seatsMax")

        seats_min = d.pop("seatsMin")

        teams = d.pop("teams")

        _monthly = d.pop("monthly", UNSET)
        monthly: EarnPrice | Unset
        if isinstance(_monthly,  Unset):
            monthly = UNSET
        else:
            monthly = EarnPrice.from_dict(_monthly)




        _yearly = d.pop("yearly", UNSET)
        yearly: EarnPrice | Unset
        if isinstance(_yearly,  Unset):
            yearly = UNSET
        else:
            yearly = EarnPrice.from_dict(_yearly)




        earn_group_prices = cls(
            seats=seats,
            seats_max=seats_max,
            seats_min=seats_min,
            teams=teams,
            monthly=monthly,
            yearly=yearly,
        )

        return earn_group_prices

