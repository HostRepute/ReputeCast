from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="EarnPagesSettings")



@_attrs_define
class EarnPagesSettings:
    """ 
        Attributes:
            can_edit (bool):
            cancel_host (str):
            gift_host (str):
            helpers (list[str]):
            support_email (str):
            terms_url (str):
            upgrade_host (str):
            upgrade_live (bool):
            upgrade_url (str):
            visits (int):
     """

    can_edit: bool
    cancel_host: str
    gift_host: str
    helpers: list[str]
    support_email: str
    terms_url: str
    upgrade_host: str
    upgrade_live: bool
    upgrade_url: str
    visits: int





    def to_dict(self) -> dict[str, Any]:
        can_edit = self.can_edit

        cancel_host = self.cancel_host

        gift_host = self.gift_host

        helpers = self.helpers



        support_email = self.support_email

        terms_url = self.terms_url

        upgrade_host = self.upgrade_host

        upgrade_live = self.upgrade_live

        upgrade_url = self.upgrade_url

        visits = self.visits


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canEdit": can_edit,
            "cancelHost": cancel_host,
            "giftHost": gift_host,
            "helpers": helpers,
            "supportEmail": support_email,
            "termsUrl": terms_url,
            "upgradeHost": upgrade_host,
            "upgradeLive": upgrade_live,
            "upgradeUrl": upgrade_url,
            "visits": visits,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        can_edit = d.pop("canEdit")

        cancel_host = d.pop("cancelHost")

        gift_host = d.pop("giftHost")

        helpers = cast(list[str], d.pop("helpers"))


        support_email = d.pop("supportEmail")

        terms_url = d.pop("termsUrl")

        upgrade_host = d.pop("upgradeHost")

        upgrade_live = d.pop("upgradeLive")

        upgrade_url = d.pop("upgradeUrl")

        visits = d.pop("visits")

        earn_pages_settings = cls(
            can_edit=can_edit,
            cancel_host=cancel_host,
            gift_host=gift_host,
            helpers=helpers,
            support_email=support_email,
            terms_url=terms_url,
            upgrade_host=upgrade_host,
            upgrade_live=upgrade_live,
            upgrade_url=upgrade_url,
            visits=visits,
        )

        return earn_pages_settings

