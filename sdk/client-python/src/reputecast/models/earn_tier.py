from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_tier_sync_status import EarnTierSyncStatus
from ..models.earn_tier_trial_days import EarnTierTrialDays
from ..models.earn_tier_visibility import EarnTierVisibility
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.earn_group_prices import EarnGroupPrices
  from ..models.earn_list_ref import EarnListRef
  from ..models.earn_price import EarnPrice
  from ..models.earn_sync_step import EarnSyncStep





T = TypeVar("T", bound="EarnTier")



@_attrs_define
class EarnTier:
    """ 
        Attributes:
            access_rank (int):
            benefits (list[str]):
            can_delete (bool):
            created_at (datetime.datetime):
            description (str):
            discounted_yearly (int):
            hidden_link (str):
            id (str):
            includes (list[str]):
            lists (list[EarnListRef]):
            mrr_cents (int):
            name (str):
            paying (int):
            paying_monthly (int):
            paying_yearly (int):
            prices (list[EarnPrice]):
            short_name (str):
            slug (str):
            sync_status (EarnTierSyncStatus):
            sync_steps (list[EarnSyncStep]):
            trial_days (EarnTierTrialDays):
            trialing (int):
            visibility (EarnTierVisibility):
            archived_at (datetime.datetime | Unset):
            group (EarnGroupPrices | Unset):
            highlight (str | Unset):
            imported_at (datetime.datetime | Unset):
            imported_from (str | Unset):
            monthly (EarnPrice | Unset):
            origin (str | Unset):
            product_id (str | Unset):
            product_url (str | Unset):
            synced_at (datetime.datetime | Unset):
            yearly (EarnPrice | Unset):
     """

    access_rank: int
    benefits: list[str]
    can_delete: bool
    created_at: datetime.datetime
    description: str
    discounted_yearly: int
    hidden_link: str
    id: str
    includes: list[str]
    lists: list[EarnListRef]
    mrr_cents: int
    name: str
    paying: int
    paying_monthly: int
    paying_yearly: int
    prices: list[EarnPrice]
    short_name: str
    slug: str
    sync_status: EarnTierSyncStatus
    sync_steps: list[EarnSyncStep]
    trial_days: EarnTierTrialDays
    trialing: int
    visibility: EarnTierVisibility
    archived_at: datetime.datetime | Unset = UNSET
    group: EarnGroupPrices | Unset = UNSET
    highlight: str | Unset = UNSET
    imported_at: datetime.datetime | Unset = UNSET
    imported_from: str | Unset = UNSET
    monthly: EarnPrice | Unset = UNSET
    origin: str | Unset = UNSET
    product_id: str | Unset = UNSET
    product_url: str | Unset = UNSET
    synced_at: datetime.datetime | Unset = UNSET
    yearly: EarnPrice | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_group_prices import EarnGroupPrices # noqa: PLC0415
        from ..models.earn_list_ref import EarnListRef # noqa: PLC0415
        from ..models.earn_price import EarnPrice # noqa: PLC0415
        from ..models.earn_sync_step import EarnSyncStep # noqa: PLC0415
        access_rank = self.access_rank

        benefits = self.benefits



        can_delete = self.can_delete

        created_at = self.created_at.isoformat()

        description = self.description

        discounted_yearly = self.discounted_yearly

        hidden_link = self.hidden_link

        id = self.id

        includes = self.includes



        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        mrr_cents = self.mrr_cents

        name = self.name

        paying = self.paying

        paying_monthly = self.paying_monthly

        paying_yearly = self.paying_yearly

        prices = []
        for prices_item_data in self.prices:
            prices_item = prices_item_data.to_dict()
            prices.append(prices_item)



        short_name = self.short_name

        slug = self.slug

        sync_status = self.sync_status.value

        sync_steps = []
        for sync_steps_item_data in self.sync_steps:
            sync_steps_item = sync_steps_item_data.to_dict()
            sync_steps.append(sync_steps_item)



        trial_days = self.trial_days.value

        trialing = self.trialing

        visibility = self.visibility.value

        archived_at: str | Unset = UNSET
        if not isinstance(self.archived_at, Unset):
            archived_at = self.archived_at.isoformat()

        group: dict[str, Any] | Unset = UNSET
        if not isinstance(self.group, Unset):
            group = self.group.to_dict()

        highlight = self.highlight

        imported_at: str | Unset = UNSET
        if not isinstance(self.imported_at, Unset):
            imported_at = self.imported_at.isoformat()

        imported_from = self.imported_from

        monthly: dict[str, Any] | Unset = UNSET
        if not isinstance(self.monthly, Unset):
            monthly = self.monthly.to_dict()

        origin = self.origin

        product_id = self.product_id

        product_url = self.product_url

        synced_at: str | Unset = UNSET
        if not isinstance(self.synced_at, Unset):
            synced_at = self.synced_at.isoformat()

        yearly: dict[str, Any] | Unset = UNSET
        if not isinstance(self.yearly, Unset):
            yearly = self.yearly.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accessRank": access_rank,
            "benefits": benefits,
            "canDelete": can_delete,
            "createdAt": created_at,
            "description": description,
            "discountedYearly": discounted_yearly,
            "hiddenLink": hidden_link,
            "id": id,
            "includes": includes,
            "lists": lists,
            "mrrCents": mrr_cents,
            "name": name,
            "paying": paying,
            "payingMonthly": paying_monthly,
            "payingYearly": paying_yearly,
            "prices": prices,
            "shortName": short_name,
            "slug": slug,
            "syncStatus": sync_status,
            "syncSteps": sync_steps,
            "trialDays": trial_days,
            "trialing": trialing,
            "visibility": visibility,
        })
        if archived_at is not UNSET:
            field_dict["archivedAt"] = archived_at
        if group is not UNSET:
            field_dict["group"] = group
        if highlight is not UNSET:
            field_dict["highlight"] = highlight
        if imported_at is not UNSET:
            field_dict["importedAt"] = imported_at
        if imported_from is not UNSET:
            field_dict["importedFrom"] = imported_from
        if monthly is not UNSET:
            field_dict["monthly"] = monthly
        if origin is not UNSET:
            field_dict["origin"] = origin
        if product_id is not UNSET:
            field_dict["productId"] = product_id
        if product_url is not UNSET:
            field_dict["productUrl"] = product_url
        if synced_at is not UNSET:
            field_dict["syncedAt"] = synced_at
        if yearly is not UNSET:
            field_dict["yearly"] = yearly

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_group_prices import EarnGroupPrices # noqa: PLC0415
        from ..models.earn_list_ref import EarnListRef # noqa: PLC0415
        from ..models.earn_price import EarnPrice # noqa: PLC0415
        from ..models.earn_sync_step import EarnSyncStep # noqa: PLC0415
        d = dict(src_dict)
        access_rank = d.pop("accessRank")

        benefits = cast(list[str], d.pop("benefits"))


        can_delete = d.pop("canDelete")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        description = d.pop("description")

        discounted_yearly = d.pop("discountedYearly")

        hidden_link = d.pop("hiddenLink")

        id = d.pop("id")

        includes = cast(list[str], d.pop("includes"))


        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = EarnListRef.from_dict(lists_item_data)



            lists.append(lists_item)


        mrr_cents = d.pop("mrrCents")

        name = d.pop("name")

        paying = d.pop("paying")

        paying_monthly = d.pop("payingMonthly")

        paying_yearly = d.pop("payingYearly")

        prices = []
        _prices = d.pop("prices")
        for prices_item_data in (_prices):
            prices_item = EarnPrice.from_dict(prices_item_data)



            prices.append(prices_item)


        short_name = d.pop("shortName")

        slug = d.pop("slug")

        sync_status = EarnTierSyncStatus(d.pop("syncStatus"))




        sync_steps = []
        _sync_steps = d.pop("syncSteps")
        for sync_steps_item_data in (_sync_steps):
            sync_steps_item = EarnSyncStep.from_dict(sync_steps_item_data)



            sync_steps.append(sync_steps_item)


        trial_days = EarnTierTrialDays(d.pop("trialDays"))




        trialing = d.pop("trialing")

        visibility = EarnTierVisibility(d.pop("visibility"))




        _archived_at = d.pop("archivedAt", UNSET)
        archived_at: datetime.datetime | Unset
        if isinstance(_archived_at,  Unset):
            archived_at = UNSET
        else:
            archived_at = datetime.datetime.fromisoformat(_archived_at)




        _group = d.pop("group", UNSET)
        group: EarnGroupPrices | Unset
        if isinstance(_group,  Unset):
            group = UNSET
        else:
            group = EarnGroupPrices.from_dict(_group)




        highlight = d.pop("highlight", UNSET)

        _imported_at = d.pop("importedAt", UNSET)
        imported_at: datetime.datetime | Unset
        if isinstance(_imported_at,  Unset):
            imported_at = UNSET
        else:
            imported_at = datetime.datetime.fromisoformat(_imported_at)




        imported_from = d.pop("importedFrom", UNSET)

        _monthly = d.pop("monthly", UNSET)
        monthly: EarnPrice | Unset
        if isinstance(_monthly,  Unset):
            monthly = UNSET
        else:
            monthly = EarnPrice.from_dict(_monthly)




        origin = d.pop("origin", UNSET)

        product_id = d.pop("productId", UNSET)

        product_url = d.pop("productUrl", UNSET)

        _synced_at = d.pop("syncedAt", UNSET)
        synced_at: datetime.datetime | Unset
        if isinstance(_synced_at,  Unset):
            synced_at = UNSET
        else:
            synced_at = datetime.datetime.fromisoformat(_synced_at)




        _yearly = d.pop("yearly", UNSET)
        yearly: EarnPrice | Unset
        if isinstance(_yearly,  Unset):
            yearly = UNSET
        else:
            yearly = EarnPrice.from_dict(_yearly)




        earn_tier = cls(
            access_rank=access_rank,
            benefits=benefits,
            can_delete=can_delete,
            created_at=created_at,
            description=description,
            discounted_yearly=discounted_yearly,
            hidden_link=hidden_link,
            id=id,
            includes=includes,
            lists=lists,
            mrr_cents=mrr_cents,
            name=name,
            paying=paying,
            paying_monthly=paying_monthly,
            paying_yearly=paying_yearly,
            prices=prices,
            short_name=short_name,
            slug=slug,
            sync_status=sync_status,
            sync_steps=sync_steps,
            trial_days=trial_days,
            trialing=trialing,
            visibility=visibility,
            archived_at=archived_at,
            group=group,
            highlight=highlight,
            imported_at=imported_at,
            imported_from=imported_from,
            monthly=monthly,
            origin=origin,
            product_id=product_id,
            product_url=product_url,
            synced_at=synced_at,
            yearly=yearly,
        )

        return earn_tier

