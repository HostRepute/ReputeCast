from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.live_rates_request_rates import LiveRatesRequestRates





T = TypeVar("T", bound="LiveRatesRequest")



@_attrs_define
class LiveRatesRequest:
    """ 
        Attributes:
            rates (LiveRatesRequestRates):
     """

    rates: LiveRatesRequestRates





    def to_dict(self) -> dict[str, Any]:
        from ..models.live_rates_request_rates import LiveRatesRequestRates # noqa: PLC0415
        rates = self.rates.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "rates": rates,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.live_rates_request_rates import LiveRatesRequestRates # noqa: PLC0415
        d = dict(src_dict)
        rates = LiveRatesRequestRates.from_dict(d.pop("rates"))




        live_rates_request = cls(
            rates=rates,
        )

        return live_rates_request

