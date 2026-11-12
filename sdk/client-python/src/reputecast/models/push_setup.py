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
  from ..models.push_keys_view import PushKeysView
  from ..models.push_month import PushMonth
  from ..models.push_prompt_card import PushPromptCard
  from ..models.push_site import PushSite
  from ..models.push_snippet import PushSnippet
  from ..models.push_subscribers import PushSubscribers





T = TypeVar("T", bound="PushSetup")



@_attrs_define
class PushSetup:
    """ 
        Attributes:
            can_edit_prompts (bool):
            can_manage (bool):
            cap (int):
            helpers (list[str]):
            month (PushMonth):
            on (bool):
            prompts (list[PushPromptCard]):
            quiet (bool):
            sites (list[PushSite]):
            snippet (PushSnippet):
            subscribers (PushSubscribers):
            window (str):
            checked_at (datetime.datetime | Unset):
            keys (PushKeysView | Unset):
            website (str | Unset):
     """

    can_edit_prompts: bool
    can_manage: bool
    cap: int
    helpers: list[str]
    month: PushMonth
    on: bool
    prompts: list[PushPromptCard]
    quiet: bool
    sites: list[PushSite]
    snippet: PushSnippet
    subscribers: PushSubscribers
    window: str
    checked_at: datetime.datetime | Unset = UNSET
    keys: PushKeysView | Unset = UNSET
    website: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.push_keys_view import PushKeysView # noqa: PLC0415
        from ..models.push_month import PushMonth # noqa: PLC0415
        from ..models.push_prompt_card import PushPromptCard # noqa: PLC0415
        from ..models.push_site import PushSite # noqa: PLC0415
        from ..models.push_snippet import PushSnippet # noqa: PLC0415
        from ..models.push_subscribers import PushSubscribers # noqa: PLC0415
        can_edit_prompts = self.can_edit_prompts

        can_manage = self.can_manage

        cap = self.cap

        helpers = self.helpers



        month = self.month.to_dict()

        on = self.on

        prompts = []
        for prompts_item_data in self.prompts:
            prompts_item = prompts_item_data.to_dict()
            prompts.append(prompts_item)



        quiet = self.quiet

        sites = []
        for sites_item_data in self.sites:
            sites_item = sites_item_data.to_dict()
            sites.append(sites_item)



        snippet = self.snippet.to_dict()

        subscribers = self.subscribers.to_dict()

        window = self.window

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        keys: dict[str, Any] | Unset = UNSET
        if not isinstance(self.keys, Unset):
            keys = self.keys.to_dict()

        website = self.website


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canEditPrompts": can_edit_prompts,
            "canManage": can_manage,
            "cap": cap,
            "helpers": helpers,
            "month": month,
            "on": on,
            "prompts": prompts,
            "quiet": quiet,
            "sites": sites,
            "snippet": snippet,
            "subscribers": subscribers,
            "window": window,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if keys is not UNSET:
            field_dict["keys"] = keys
        if website is not UNSET:
            field_dict["website"] = website

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.push_keys_view import PushKeysView # noqa: PLC0415
        from ..models.push_month import PushMonth # noqa: PLC0415
        from ..models.push_prompt_card import PushPromptCard # noqa: PLC0415
        from ..models.push_site import PushSite # noqa: PLC0415
        from ..models.push_snippet import PushSnippet # noqa: PLC0415
        from ..models.push_subscribers import PushSubscribers # noqa: PLC0415
        d = dict(src_dict)
        can_edit_prompts = d.pop("canEditPrompts")

        can_manage = d.pop("canManage")

        cap = d.pop("cap")

        helpers = cast(list[str], d.pop("helpers"))


        month = PushMonth.from_dict(d.pop("month"))




        on = d.pop("on")

        prompts = []
        _prompts = d.pop("prompts")
        for prompts_item_data in (_prompts):
            prompts_item = PushPromptCard.from_dict(prompts_item_data)



            prompts.append(prompts_item)


        quiet = d.pop("quiet")

        sites = []
        _sites = d.pop("sites")
        for sites_item_data in (_sites):
            sites_item = PushSite.from_dict(sites_item_data)



            sites.append(sites_item)


        snippet = PushSnippet.from_dict(d.pop("snippet"))




        subscribers = PushSubscribers.from_dict(d.pop("subscribers"))




        window = d.pop("window")

        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        _keys = d.pop("keys", UNSET)
        keys: PushKeysView | Unset
        if isinstance(_keys,  Unset):
            keys = UNSET
        else:
            keys = PushKeysView.from_dict(_keys)




        website = d.pop("website", UNSET)

        push_setup = cls(
            can_edit_prompts=can_edit_prompts,
            can_manage=can_manage,
            cap=cap,
            helpers=helpers,
            month=month,
            on=on,
            prompts=prompts,
            quiet=quiet,
            sites=sites,
            snippet=snippet,
            subscribers=subscribers,
            window=window,
            checked_at=checked_at,
            keys=keys,
            website=website,
        )

        return push_setup

