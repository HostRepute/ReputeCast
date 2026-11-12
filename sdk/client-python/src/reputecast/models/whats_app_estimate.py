from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.whats_app_cost import WhatsAppCost





T = TypeVar("T", bound="WhatsAppEstimate")



@_attrs_define
class WhatsAppEstimate:
    """ 
        Attributes:
            connected (bool):
            consent (int):
            cost (WhatsAppCost):
            open_ (int):
            quiet (bool):
            window (str):
     """

    connected: bool
    consent: int
    cost: WhatsAppCost
    open_: int
    quiet: bool
    window: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.whats_app_cost import WhatsAppCost # noqa: PLC0415
        connected = self.connected

        consent = self.consent

        cost = self.cost.to_dict()

        open_ = self.open_

        quiet = self.quiet

        window = self.window


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "connected": connected,
            "consent": consent,
            "cost": cost,
            "open": open_,
            "quiet": quiet,
            "window": window,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.whats_app_cost import WhatsAppCost # noqa: PLC0415
        d = dict(src_dict)
        connected = d.pop("connected")

        consent = d.pop("consent")

        cost = WhatsAppCost.from_dict(d.pop("cost"))




        open_ = d.pop("open")

        quiet = d.pop("quiet")

        window = d.pop("window")

        whats_app_estimate = cls(
            connected=connected,
            consent=consent,
            cost=cost,
            open_=open_,
            quiet=quiet,
            window=window,
        )

        return whats_app_estimate

