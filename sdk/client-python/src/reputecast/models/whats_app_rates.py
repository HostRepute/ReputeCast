from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.whats_app_rate import WhatsAppRate





T = TypeVar("T", bound="WhatsAppRates")



@_attrs_define
class WhatsAppRates:
    """ 
        Attributes:
            as_of (datetime.datetime):
            can_manage (bool):
            rates (list[WhatsAppRate]):
            edited_at (datetime.datetime | Unset):
            edited_by (str | Unset):
     """

    as_of: datetime.datetime
    can_manage: bool
    rates: list[WhatsAppRate]
    edited_at: datetime.datetime | Unset = UNSET
    edited_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.whats_app_rate import WhatsAppRate # noqa: PLC0415
        as_of = self.as_of.isoformat()

        can_manage = self.can_manage

        rates = []
        for rates_item_data in self.rates:
            rates_item = rates_item_data.to_dict()
            rates.append(rates_item)



        edited_at: str | Unset = UNSET
        if not isinstance(self.edited_at, Unset):
            edited_at = self.edited_at.isoformat()

        edited_by = self.edited_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "asOf": as_of,
            "canManage": can_manage,
            "rates": rates,
        })
        if edited_at is not UNSET:
            field_dict["editedAt"] = edited_at
        if edited_by is not UNSET:
            field_dict["editedBy"] = edited_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.whats_app_rate import WhatsAppRate # noqa: PLC0415
        d = dict(src_dict)
        as_of = datetime.datetime.fromisoformat(d.pop("asOf"))




        can_manage = d.pop("canManage")

        rates = []
        _rates = d.pop("rates")
        for rates_item_data in (_rates):
            rates_item = WhatsAppRate.from_dict(rates_item_data)



            rates.append(rates_item)


        _edited_at = d.pop("editedAt", UNSET)
        edited_at: datetime.datetime | Unset
        if isinstance(_edited_at,  Unset):
            edited_at = UNSET
        else:
            edited_at = datetime.datetime.fromisoformat(_edited_at)




        edited_by = d.pop("editedBy", UNSET)

        whats_app_rates = cls(
            as_of=as_of,
            can_manage=can_manage,
            rates=rates,
            edited_at=edited_at,
            edited_by=edited_by,
        )

        return whats_app_rates

