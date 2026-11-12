from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.coupon_pool_source import CouponPoolSource
from ..models.coupon_pool_status import CouponPoolStatus
from ..types import UNSET, Unset
from typing import cast
from uuid import UUID
import datetime

if TYPE_CHECKING:
  from ..models.coupon_batch import CouponBatch
  from ..models.coupon_pool_settings import CouponPoolSettings
  from ..models.coupon_pool_store import CouponPoolStore
  from ..models.coupon_pool_tokens import CouponPoolTokens
  from ..models.coupon_pool_use import CouponPoolUse
  from ..models.coupon_pool_waiting import CouponPoolWaiting





T = TypeVar("T", bound="CouponPool")



@_attrs_define
class CouponPool:
    """ 
        Attributes:
            batches (list[CouponBatch]):
            created_at (datetime.datetime):
            created_by (str): Who created the pool, by name
            currency (str):
            free (int):
            helpers (list[str]):
            id (UUID):
            low (bool): Free codes are under the low mark
            low_at (int):
            name (str):
            one_per_contact (bool):
            orders (int):
            prefix (str):
            redeemed (int):
            refill (bool):
            removed (int):
            revenue_cents (int): Orders that used the pool's codes: refunds taken off, cancelled and test orders left out
            sample_url (str):
            sent (int):
            settings (CouponPoolSettings):
            slug (str): The name templates use: {{ coupons.<slug>.code }}
            source (CouponPoolSource):
            status (CouponPoolStatus):
            tokens (CouponPoolTokens):
            total (int):
            updated_at (datetime.datetime):
            used_in (list[CouponPoolUse]):
            waiting (list[CouponPoolWaiting]):
            batch (CouponBatch | Unset):
            first_sent_at (datetime.datetime | Unset):
            refill_checked_at (datetime.datetime | Unset):
            running (CouponBatch | Unset):
            store (CouponPoolStore | Unset):
     """

    batches: list[CouponBatch]
    created_at: datetime.datetime
    created_by: str
    currency: str
    free: int
    helpers: list[str]
    id: UUID
    low: bool
    low_at: int
    name: str
    one_per_contact: bool
    orders: int
    prefix: str
    redeemed: int
    refill: bool
    removed: int
    revenue_cents: int
    sample_url: str
    sent: int
    settings: CouponPoolSettings
    slug: str
    source: CouponPoolSource
    status: CouponPoolStatus
    tokens: CouponPoolTokens
    total: int
    updated_at: datetime.datetime
    used_in: list[CouponPoolUse]
    waiting: list[CouponPoolWaiting]
    batch: CouponBatch | Unset = UNSET
    first_sent_at: datetime.datetime | Unset = UNSET
    refill_checked_at: datetime.datetime | Unset = UNSET
    running: CouponBatch | Unset = UNSET
    store: CouponPoolStore | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.coupon_batch import CouponBatch # noqa: PLC0415
        from ..models.coupon_pool_settings import CouponPoolSettings # noqa: PLC0415
        from ..models.coupon_pool_store import CouponPoolStore # noqa: PLC0415
        from ..models.coupon_pool_tokens import CouponPoolTokens # noqa: PLC0415
        from ..models.coupon_pool_use import CouponPoolUse # noqa: PLC0415
        from ..models.coupon_pool_waiting import CouponPoolWaiting # noqa: PLC0415
        batches = []
        for batches_item_data in self.batches:
            batches_item = batches_item_data.to_dict()
            batches.append(batches_item)



        created_at = self.created_at.isoformat()

        created_by = self.created_by

        currency = self.currency

        free = self.free

        helpers = self.helpers



        id = str(self.id)

        low = self.low

        low_at = self.low_at

        name = self.name

        one_per_contact = self.one_per_contact

        orders = self.orders

        prefix = self.prefix

        redeemed = self.redeemed

        refill = self.refill

        removed = self.removed

        revenue_cents = self.revenue_cents

        sample_url = self.sample_url

        sent = self.sent

        settings = self.settings.to_dict()

        slug = self.slug

        source = self.source.value

        status = self.status.value

        tokens = self.tokens.to_dict()

        total = self.total

        updated_at = self.updated_at.isoformat()

        used_in = []
        for used_in_item_data in self.used_in:
            used_in_item = used_in_item_data.to_dict()
            used_in.append(used_in_item)



        waiting = []
        for waiting_item_data in self.waiting:
            waiting_item = waiting_item_data.to_dict()
            waiting.append(waiting_item)



        batch: dict[str, Any] | Unset = UNSET
        if not isinstance(self.batch, Unset):
            batch = self.batch.to_dict()

        first_sent_at: str | Unset = UNSET
        if not isinstance(self.first_sent_at, Unset):
            first_sent_at = self.first_sent_at.isoformat()

        refill_checked_at: str | Unset = UNSET
        if not isinstance(self.refill_checked_at, Unset):
            refill_checked_at = self.refill_checked_at.isoformat()

        running: dict[str, Any] | Unset = UNSET
        if not isinstance(self.running, Unset):
            running = self.running.to_dict()

        store: dict[str, Any] | Unset = UNSET
        if not isinstance(self.store, Unset):
            store = self.store.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "batches": batches,
            "createdAt": created_at,
            "createdBy": created_by,
            "currency": currency,
            "free": free,
            "helpers": helpers,
            "id": id,
            "low": low,
            "lowAt": low_at,
            "name": name,
            "onePerContact": one_per_contact,
            "orders": orders,
            "prefix": prefix,
            "redeemed": redeemed,
            "refill": refill,
            "removed": removed,
            "revenueCents": revenue_cents,
            "sampleUrl": sample_url,
            "sent": sent,
            "settings": settings,
            "slug": slug,
            "source": source,
            "status": status,
            "tokens": tokens,
            "total": total,
            "updatedAt": updated_at,
            "usedIn": used_in,
            "waiting": waiting,
        })
        if batch is not UNSET:
            field_dict["batch"] = batch
        if first_sent_at is not UNSET:
            field_dict["firstSentAt"] = first_sent_at
        if refill_checked_at is not UNSET:
            field_dict["refillCheckedAt"] = refill_checked_at
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
        from ..models.coupon_pool_tokens import CouponPoolTokens # noqa: PLC0415
        from ..models.coupon_pool_use import CouponPoolUse # noqa: PLC0415
        from ..models.coupon_pool_waiting import CouponPoolWaiting # noqa: PLC0415
        d = dict(src_dict)
        batches = []
        _batches = d.pop("batches")
        for batches_item_data in (_batches):
            batches_item = CouponBatch.from_dict(batches_item_data)



            batches.append(batches_item)


        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        created_by = d.pop("createdBy")

        currency = d.pop("currency")

        free = d.pop("free")

        helpers = cast(list[str], d.pop("helpers"))


        id = UUID(d.pop("id"))




        low = d.pop("low")

        low_at = d.pop("lowAt")

        name = d.pop("name")

        one_per_contact = d.pop("onePerContact")

        orders = d.pop("orders")

        prefix = d.pop("prefix")

        redeemed = d.pop("redeemed")

        refill = d.pop("refill")

        removed = d.pop("removed")

        revenue_cents = d.pop("revenueCents")

        sample_url = d.pop("sampleUrl")

        sent = d.pop("sent")

        settings = CouponPoolSettings.from_dict(d.pop("settings"))




        slug = d.pop("slug")

        source = CouponPoolSource(d.pop("source"))




        status = CouponPoolStatus(d.pop("status"))




        tokens = CouponPoolTokens.from_dict(d.pop("tokens"))




        total = d.pop("total")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        used_in = []
        _used_in = d.pop("usedIn")
        for used_in_item_data in (_used_in):
            used_in_item = CouponPoolUse.from_dict(used_in_item_data)



            used_in.append(used_in_item)


        waiting = []
        _waiting = d.pop("waiting")
        for waiting_item_data in (_waiting):
            waiting_item = CouponPoolWaiting.from_dict(waiting_item_data)



            waiting.append(waiting_item)


        _batch = d.pop("batch", UNSET)
        batch: CouponBatch | Unset
        if isinstance(_batch,  Unset):
            batch = UNSET
        else:
            batch = CouponBatch.from_dict(_batch)




        _first_sent_at = d.pop("firstSentAt", UNSET)
        first_sent_at: datetime.datetime | Unset
        if isinstance(_first_sent_at,  Unset):
            first_sent_at = UNSET
        else:
            first_sent_at = datetime.datetime.fromisoformat(_first_sent_at)




        _refill_checked_at = d.pop("refillCheckedAt", UNSET)
        refill_checked_at: datetime.datetime | Unset
        if isinstance(_refill_checked_at,  Unset):
            refill_checked_at = UNSET
        else:
            refill_checked_at = datetime.datetime.fromisoformat(_refill_checked_at)




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




        coupon_pool = cls(
            batches=batches,
            created_at=created_at,
            created_by=created_by,
            currency=currency,
            free=free,
            helpers=helpers,
            id=id,
            low=low,
            low_at=low_at,
            name=name,
            one_per_contact=one_per_contact,
            orders=orders,
            prefix=prefix,
            redeemed=redeemed,
            refill=refill,
            removed=removed,
            revenue_cents=revenue_cents,
            sample_url=sample_url,
            sent=sent,
            settings=settings,
            slug=slug,
            source=source,
            status=status,
            tokens=tokens,
            total=total,
            updated_at=updated_at,
            used_in=used_in,
            waiting=waiting,
            batch=batch,
            first_sent_at=first_sent_at,
            refill_checked_at=refill_checked_at,
            running=running,
            store=store,
        )

        return coupon_pool

