from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.coupon_pool_settings_kind import CouponPoolSettingsKind
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="CouponPoolSettings")



@_attrs_define
class CouponPoolSettings:
    """ 
        Attributes:
            apply_url (str | Unset): Upload pools: the address that applies a code, with {code} where it goes
            collection_ids (list[str] | Unset): Store pools: the collections the discount applies to; none is the whole
                store
            ends_days (int | Unset): Store pools: the discount ends this many days after the pool was created; 0 never
            kind (CouponPoolSettingsKind | Unset): Store pools: what the discount takes off
            minimum_cents (int | Unset): Store pools: the smallest order the code works on, in cents
            value (int | Unset): Percent off (1–100) for percentage, cents off for fixed_amount
     """

    apply_url: str | Unset = UNSET
    collection_ids: list[str] | Unset = UNSET
    ends_days: int | Unset = UNSET
    kind: CouponPoolSettingsKind | Unset = UNSET
    minimum_cents: int | Unset = UNSET
    value: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        apply_url = self.apply_url

        collection_ids: list[str] | Unset = UNSET
        if not isinstance(self.collection_ids, Unset):
            collection_ids = self.collection_ids



        ends_days = self.ends_days

        kind: str | Unset = UNSET
        if not isinstance(self.kind, Unset):
            kind = self.kind.value


        minimum_cents = self.minimum_cents

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if apply_url is not UNSET:
            field_dict["applyUrl"] = apply_url
        if collection_ids is not UNSET:
            field_dict["collectionIds"] = collection_ids
        if ends_days is not UNSET:
            field_dict["endsDays"] = ends_days
        if kind is not UNSET:
            field_dict["kind"] = kind
        if minimum_cents is not UNSET:
            field_dict["minimumCents"] = minimum_cents
        if value is not UNSET:
            field_dict["value"] = value

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        apply_url = d.pop("applyUrl", UNSET)

        collection_ids = cast(list[str], d.pop("collectionIds", UNSET))


        ends_days = d.pop("endsDays", UNSET)

        _kind = d.pop("kind", UNSET)
        kind: CouponPoolSettingsKind | Unset
        if isinstance(_kind,  Unset):
            kind = UNSET
        else:
            kind = CouponPoolSettingsKind(_kind)




        minimum_cents = d.pop("minimumCents", UNSET)

        value = d.pop("value", UNSET)

        coupon_pool_settings = cls(
            apply_url=apply_url,
            collection_ids=collection_ids,
            ends_days=ends_days,
            kind=kind,
            minimum_cents=minimum_cents,
            value=value,
        )

        return coupon_pool_settings

