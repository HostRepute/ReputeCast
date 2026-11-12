from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.approval_share_color import ApprovalShareColor
from ..types import UNSET, Unset






T = TypeVar("T", bound="ApprovalShare")



@_attrs_define
class ApprovalShare:
    """ 
        Attributes:
            color (ApprovalShareColor):
            cost (float):
            name (str):
            price_label (str):
            provider_id (str):
            recipients (int):
            short (str):
            status (str):
            status_label (str):
            label (str | Unset):
     """

    color: ApprovalShareColor
    cost: float
    name: str
    price_label: str
    provider_id: str
    recipients: int
    short: str
    status: str
    status_label: str
    label: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        color = self.color.value

        cost = self.cost

        name = self.name

        price_label = self.price_label

        provider_id = self.provider_id

        recipients = self.recipients

        short = self.short

        status = self.status

        status_label = self.status_label

        label = self.label


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "color": color,
            "cost": cost,
            "name": name,
            "priceLabel": price_label,
            "providerId": provider_id,
            "recipients": recipients,
            "short": short,
            "status": status,
            "statusLabel": status_label,
        })
        if label is not UNSET:
            field_dict["label"] = label

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        color = ApprovalShareColor(d.pop("color"))




        cost = d.pop("cost")

        name = d.pop("name")

        price_label = d.pop("priceLabel")

        provider_id = d.pop("providerId")

        recipients = d.pop("recipients")

        short = d.pop("short")

        status = d.pop("status")

        status_label = d.pop("statusLabel")

        label = d.pop("label", UNSET)

        approval_share = cls(
            color=color,
            cost=cost,
            name=name,
            price_label=price_label,
            provider_id=provider_id,
            recipients=recipients,
            short=short,
            status=status,
            status_label=status_label,
            label=label,
        )

        return approval_share

