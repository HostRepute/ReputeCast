from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.coupon_source_platform import CouponSourcePlatform
from ..models.coupon_source_reason import CouponSourceReason
from ..types import UNSET, Unset






T = TypeVar("T", bound="CouponSource")



@_attrs_define
class CouponSource:
    """ 
        Attributes:
            can (bool):
            currency (str):
            domain (str):
            name (str):
            platform (CouponSourcePlatform):
            store_id (str):
            detail (str | Unset):
            reason (CouponSourceReason | Unset):
     """

    can: bool
    currency: str
    domain: str
    name: str
    platform: CouponSourcePlatform
    store_id: str
    detail: str | Unset = UNSET
    reason: CouponSourceReason | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        can = self.can

        currency = self.currency

        domain = self.domain

        name = self.name

        platform = self.platform.value

        store_id = self.store_id

        detail = self.detail

        reason: str | Unset = UNSET
        if not isinstance(self.reason, Unset):
            reason = self.reason.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "can": can,
            "currency": currency,
            "domain": domain,
            "name": name,
            "platform": platform,
            "storeId": store_id,
        })
        if detail is not UNSET:
            field_dict["detail"] = detail
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        can = d.pop("can")

        currency = d.pop("currency")

        domain = d.pop("domain")

        name = d.pop("name")

        platform = CouponSourcePlatform(d.pop("platform"))




        store_id = d.pop("storeId")

        detail = d.pop("detail", UNSET)

        _reason = d.pop("reason", UNSET)
        reason: CouponSourceReason | Unset
        if isinstance(_reason,  Unset):
            reason = UNSET
        else:
            reason = CouponSourceReason(_reason)




        coupon_source = cls(
            can=can,
            currency=currency,
            domain=domain,
            name=name,
            platform=platform,
            store_id=store_id,
            detail=detail,
            reason=reason,
        )

        return coupon_source

