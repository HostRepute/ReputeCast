from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.coupon_pool_list_item_source import CouponPoolListItemSource
from ..models.coupon_pool_list_item_status import CouponPoolListItemStatus
from ..types import UNSET, Unset
from typing import cast
from uuid import UUID
import datetime

if TYPE_CHECKING:
  from ..models.coupon_batch import CouponBatch
  from ..models.coupon_pool_settings import CouponPoolSettings
  from ..models.coupon_pool_store import CouponPoolStore





T = TypeVar("T", bound="CouponPoolListItem")



@_attrs_define
class CouponPoolListItem:
    """ 
        Attributes:
            created_at (datetime.datetime):
            currency (str):
            free (int):
            id (UUID):
            low (bool): Free codes are under the low mark
            low_at (int):
            name (str):
            one_per_contact (bool):
            prefix (str):
            redeemed (int):
            revenue_cents (int): Orders that used the pool's codes: refunds taken off, cancelled and test orders left out
            sent (int):
            settings (CouponPoolSettings):
            slug (str): The name templates use: {{ coupons.<slug>.code }}
            source (CouponPoolListItemSource):
            status (CouponPoolListItemStatus):
            total (int):
            running (CouponBatch | Unset):
            store (CouponPoolStore | Unset):
     """

    created_at: datetime.datetime
    currency: str
    free: int
    id: UUID
    low: bool
    low_at: int
    name: str
    one_per_contact: bool
    prefix: str
    redeemed: int
    revenue_cents: int
    sent: int
    settings: CouponPoolSettings
    slug: str
    source: CouponPoolListItemSource
    status: CouponPoolListItemStatus
    total: int
    running: CouponBatch | Unset = UNSET
    store: CouponPoolStore | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.coupon_batch import CouponBatch # noqa: PLC0415
        from ..models.coupon_pool_settings import CouponPoolSettings # noqa: PLC0415
        from ..models.coupon_pool_store import CouponPoolStore # noqa: PLC0415
        created_at = self.created_at.isoformat()

        currency = self.currency

        free = self.free

        id = str(self.id)

        low = self.low

        low_at = self.low_at

        name = self.name

        one_per_contact = self.one_per_contact

        prefix = self.prefix

        redeemed = self.redeemed

        revenue_cents = self.revenue_cents

        sent = self.sent

        settings = self.settings.to_dict()

        slug = self.slug

        source = self.source.value

        status = self.status.value

        total = self.total

        running: dict[str, Any] | Unset = UNSET
        if not isinstance(self.running, Unset):
            running = self.running.to_dict()

        store: dict[str, Any] | Unset = UNSET
        if not isinstance(self.store, Unset):
            store = self.store.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "createdAt": created_at,
            "currency": currency,
            "free": free,
            "id": id,
            "low": low,
            "lowAt": low_at,
            "name": name,
            "onePerContact": one_per_contact,
            "prefix": prefix,
            "redeemed": redeemed,
            "revenueCents": revenue_cents,
            "sent": sent,
            "settings": settings,
            "slug": slug,
            "source": source,
            "status": status,
            "total": total,
        })
        if running is not UNSET:
            field_dict["running"] = running
        if store is not UNSET:
            field_dict["store"] = store

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.coupon_batch import CouponBatch # noqa: PLC0415
        from ..models.coupon_pool_settings import CouponPoolSettings # noqa: PLC0415
        from ..models.coupon_pool_store import CouponPoolStore # noqa: PLC0415
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        currency = d.pop("currency")

        free = d.pop("free")

        id = UUID(d.pop("id"))




        low = d.pop("low")

        low_at = d.pop("lowAt")

        name = d.pop("name")

        one_per_contact = d.pop("onePerContact")

        prefix = d.pop("prefix")

        redeemed = d.pop("redeemed")

        revenue_cents = d.pop("revenueCents")

        sent = d.pop("sent")

        settings = CouponPoolSettings.from_dict(d.pop("settings"))




        slug = d.pop("slug")

        source = CouponPoolListItemSource(d.pop("source"))




        status = CouponPoolListItemStatus(d.pop("status"))




        total = d.pop("total")

        _running = d.pop("running", UNSET)
        running: CouponBatch | Unset
        if isinstance(_running,  Unset):
            running = UNSET
        else:
            running = CouponBatch.from_dict(_running)




        _store = d.pop("store", UNSET)
        store: CouponPoolStore | Unset
        if isinstance(_store,  Unset):
            store = UNSET
        else:
            store = CouponPoolStore.from_dict(_store)




        coupon_pool_list_item = cls(
            created_at=created_at,
            currency=currency,
            free=free,
            id=id,
            low=low,
            low_at=low_at,
            name=name,
            one_per_contact=one_per_contact,
            prefix=prefix,
            redeemed=redeemed,
            revenue_cents=revenue_cents,
            sent=sent,
            settings=settings,
            slug=slug,
            source=source,
            status=status,
            total=total,
            running=running,
            store=store,
        )

        return coupon_pool_list_item

