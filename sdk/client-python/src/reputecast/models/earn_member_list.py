from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_member_list_access_rule import EarnMemberListAccessRule
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.earn_list_ref import EarnListRef
  from ..models.earn_member_counts import EarnMemberCounts
  from ..models.earn_member_row import EarnMemberRow





T = TypeVar("T", bound="EarnMemberList")



@_attrs_define
class EarnMemberList:
    """ 
        Attributes:
            access_rule (EarnMemberListAccessRule):
            campaigns (list[EarnListRef]):
            can_export (bool):
            can_import (bool):
            can_manage (bool):
            can_see_names (bool):
            connected (bool):
            counts (EarnMemberCounts):
            currency (str):
            export_helpers (list[str]):
            helpers (list[str]):
            items (list[EarnMemberRow]):
            mrr_cents (int):
            offers (list[str]):
            platforms (list[str]):
            test_members (int):
            test_shown (int):
            tiers (list[EarnListRef]):
            total (int):
            upgrade_host (str):
            upgrade_url (str):
            moved_at (datetime.datetime | Unset):
     """

    access_rule: EarnMemberListAccessRule
    campaigns: list[EarnListRef]
    can_export: bool
    can_import: bool
    can_manage: bool
    can_see_names: bool
    connected: bool
    counts: EarnMemberCounts
    currency: str
    export_helpers: list[str]
    helpers: list[str]
    items: list[EarnMemberRow]
    mrr_cents: int
    offers: list[str]
    platforms: list[str]
    test_members: int
    test_shown: int
    tiers: list[EarnListRef]
    total: int
    upgrade_host: str
    upgrade_url: str
    moved_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_list_ref import EarnListRef # noqa: PLC0415
        from ..models.earn_member_counts import EarnMemberCounts # noqa: PLC0415
        from ..models.earn_member_row import EarnMemberRow # noqa: PLC0415
        access_rule = self.access_rule.value

        campaigns = []
        for campaigns_item_data in self.campaigns:
            campaigns_item = campaigns_item_data.to_dict()
            campaigns.append(campaigns_item)



        can_export = self.can_export

        can_import = self.can_import

        can_manage = self.can_manage

        can_see_names = self.can_see_names

        connected = self.connected

        counts = self.counts.to_dict()

        currency = self.currency

        export_helpers = self.export_helpers



        helpers = self.helpers



        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        mrr_cents = self.mrr_cents

        offers = self.offers



        platforms = self.platforms



        test_members = self.test_members

        test_shown = self.test_shown

        tiers = []
        for tiers_item_data in self.tiers:
            tiers_item = tiers_item_data.to_dict()
            tiers.append(tiers_item)



        total = self.total

        upgrade_host = self.upgrade_host

        upgrade_url = self.upgrade_url

        moved_at: str | Unset = UNSET
        if not isinstance(self.moved_at, Unset):
            moved_at = self.moved_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accessRule": access_rule,
            "campaigns": campaigns,
            "canExport": can_export,
            "canImport": can_import,
            "canManage": can_manage,
            "canSeeNames": can_see_names,
            "connected": connected,
            "counts": counts,
            "currency": currency,
            "exportHelpers": export_helpers,
            "helpers": helpers,
            "items": items,
            "mrrCents": mrr_cents,
            "offers": offers,
            "platforms": platforms,
            "testMembers": test_members,
            "testShown": test_shown,
            "tiers": tiers,
            "total": total,
            "upgradeHost": upgrade_host,
            "upgradeUrl": upgrade_url,
        })
        if moved_at is not UNSET:
            field_dict["movedAt"] = moved_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_list_ref import EarnListRef # noqa: PLC0415
        from ..models.earn_member_counts import EarnMemberCounts # noqa: PLC0415
        from ..models.earn_member_row import EarnMemberRow # noqa: PLC0415
        d = dict(src_dict)
        access_rule = EarnMemberListAccessRule(d.pop("accessRule"))




        campaigns = []
        _campaigns = d.pop("campaigns")
        for campaigns_item_data in (_campaigns):
            campaigns_item = EarnListRef.from_dict(campaigns_item_data)



            campaigns.append(campaigns_item)


        can_export = d.pop("canExport")

        can_import = d.pop("canImport")

        can_manage = d.pop("canManage")

        can_see_names = d.pop("canSeeNames")

        connected = d.pop("connected")

        counts = EarnMemberCounts.from_dict(d.pop("counts"))




        currency = d.pop("currency")

        export_helpers = cast(list[str], d.pop("exportHelpers"))


        helpers = cast(list[str], d.pop("helpers"))


        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = EarnMemberRow.from_dict(items_item_data)



            items.append(items_item)


        mrr_cents = d.pop("mrrCents")

        offers = cast(list[str], d.pop("offers"))


        platforms = cast(list[str], d.pop("platforms"))


        test_members = d.pop("testMembers")

        test_shown = d.pop("testShown")

        tiers = []
        _tiers = d.pop("tiers")
        for tiers_item_data in (_tiers):
            tiers_item = EarnListRef.from_dict(tiers_item_data)



            tiers.append(tiers_item)


        total = d.pop("total")

        upgrade_host = d.pop("upgradeHost")

        upgrade_url = d.pop("upgradeUrl")

        _moved_at = d.pop("movedAt", UNSET)
        moved_at: datetime.datetime | Unset
        if isinstance(_moved_at,  Unset):
            moved_at = UNSET
        else:
            moved_at = datetime.datetime.fromisoformat(_moved_at)




        earn_member_list = cls(
            access_rule=access_rule,
            campaigns=campaigns,
            can_export=can_export,
            can_import=can_import,
            can_manage=can_manage,
            can_see_names=can_see_names,
            connected=connected,
            counts=counts,
            currency=currency,
            export_helpers=export_helpers,
            helpers=helpers,
            items=items,
            mrr_cents=mrr_cents,
            offers=offers,
            platforms=platforms,
            test_members=test_members,
            test_shown=test_shown,
            tiers=tiers,
            total=total,
            upgrade_host=upgrade_host,
            upgrade_url=upgrade_url,
            moved_at=moved_at,
        )

        return earn_member_list

