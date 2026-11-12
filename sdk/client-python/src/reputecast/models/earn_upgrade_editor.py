from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.earn_list_ref import EarnListRef
  from ..models.earn_page_tier import EarnPageTier
  from ..models.upgrade_page import UpgradePage





T = TypeVar("T", bound="EarnUpgradeEditor")



@_attrs_define
class EarnUpgradeEditor:
    """ 
        Attributes:
            can_edit (bool):
            helpers (list[str]):
            host (str):
            lists (list[EarnListRef]):
            live (bool):
            page (UpgradePage):
            test_checkout (bool):
            tiers (list[EarnPageTier]):
            url (str):
            published_at (str | Unset):
            published_by (str | Unset):
            test_checkout_url (str | Unset):
     """

    can_edit: bool
    helpers: list[str]
    host: str
    lists: list[EarnListRef]
    live: bool
    page: UpgradePage
    test_checkout: bool
    tiers: list[EarnPageTier]
    url: str
    published_at: str | Unset = UNSET
    published_by: str | Unset = UNSET
    test_checkout_url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_list_ref import EarnListRef # noqa: PLC0415
        from ..models.earn_page_tier import EarnPageTier # noqa: PLC0415
        from ..models.upgrade_page import UpgradePage # noqa: PLC0415
        can_edit = self.can_edit

        helpers = self.helpers



        host = self.host

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        live = self.live

        page = self.page.to_dict()

        test_checkout = self.test_checkout

        tiers = []
        for tiers_item_data in self.tiers:
            tiers_item = tiers_item_data.to_dict()
            tiers.append(tiers_item)



        url = self.url

        published_at = self.published_at

        published_by = self.published_by

        test_checkout_url = self.test_checkout_url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canEdit": can_edit,
            "helpers": helpers,
            "host": host,
            "lists": lists,
            "live": live,
            "page": page,
            "testCheckout": test_checkout,
            "tiers": tiers,
            "url": url,
        })
        if published_at is not UNSET:
            field_dict["publishedAt"] = published_at
        if published_by is not UNSET:
            field_dict["publishedBy"] = published_by
        if test_checkout_url is not UNSET:
            field_dict["testCheckoutUrl"] = test_checkout_url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_list_ref import EarnListRef # noqa: PLC0415
        from ..models.earn_page_tier import EarnPageTier # noqa: PLC0415
        from ..models.upgrade_page import UpgradePage # noqa: PLC0415
        d = dict(src_dict)
        can_edit = d.pop("canEdit")

        helpers = cast(list[str], d.pop("helpers"))


        host = d.pop("host")

        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = EarnListRef.from_dict(lists_item_data)



            lists.append(lists_item)


        live = d.pop("live")

        page = UpgradePage.from_dict(d.pop("page"))




        test_checkout = d.pop("testCheckout")

        tiers = []
        _tiers = d.pop("tiers")
        for tiers_item_data in (_tiers):
            tiers_item = EarnPageTier.from_dict(tiers_item_data)



            tiers.append(tiers_item)


        url = d.pop("url")

        published_at = d.pop("publishedAt", UNSET)

        published_by = d.pop("publishedBy", UNSET)

        test_checkout_url = d.pop("testCheckoutUrl", UNSET)

        earn_upgrade_editor = cls(
            can_edit=can_edit,
            helpers=helpers,
            host=host,
            lists=lists,
            live=live,
            page=page,
            test_checkout=test_checkout,
            tiers=tiers,
            url=url,
            published_at=published_at,
            published_by=published_by,
            test_checkout_url=test_checkout_url,
        )

        return earn_upgrade_editor

