from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_prompt_status import PushPromptStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.push_consent_record import PushConsentRecord
  from ..models.push_prompt_config import PushPromptConfig
  from ..models.push_prompt_site_option import PushPromptSiteOption
  from ..models.push_prompt_stats import PushPromptStats





T = TypeVar("T", bound="PushPrompt")



@_attrs_define
class PushPrompt:
    """ 
        Attributes:
            can_edit (bool):
            changed (bool):
            draft (PushPromptConfig):
            form_id (str):
            live_version (int):
            name (str):
            push_on (bool):
            sites (list[PushPromptSiteOption]):
            stats (PushPromptStats):
            status (PushPromptStatus):
            updated_at (datetime.datetime):
            updated_by (str):
            consent (PushConsentRecord | Unset):
            icon_url (str | Unset):
            live (PushPromptConfig | Unset):
            live_since (datetime.datetime | Unset):
            site (PushPromptSiteOption | Unset):
     """

    can_edit: bool
    changed: bool
    draft: PushPromptConfig
    form_id: str
    live_version: int
    name: str
    push_on: bool
    sites: list[PushPromptSiteOption]
    stats: PushPromptStats
    status: PushPromptStatus
    updated_at: datetime.datetime
    updated_by: str
    consent: PushConsentRecord | Unset = UNSET
    icon_url: str | Unset = UNSET
    live: PushPromptConfig | Unset = UNSET
    live_since: datetime.datetime | Unset = UNSET
    site: PushPromptSiteOption | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.push_consent_record import PushConsentRecord # noqa: PLC0415
        from ..models.push_prompt_config import PushPromptConfig # noqa: PLC0415
        from ..models.push_prompt_site_option import PushPromptSiteOption # noqa: PLC0415
        from ..models.push_prompt_stats import PushPromptStats # noqa: PLC0415
        can_edit = self.can_edit

        changed = self.changed

        draft = self.draft.to_dict()

        form_id = self.form_id

        live_version = self.live_version

        name = self.name

        push_on = self.push_on

        sites = []
        for sites_item_data in self.sites:
            sites_item = sites_item_data.to_dict()
            sites.append(sites_item)



        stats = self.stats.to_dict()

        status = self.status.value

        updated_at = self.updated_at.isoformat()

        updated_by = self.updated_by

        consent: dict[str, Any] | Unset = UNSET
        if not isinstance(self.consent, Unset):
            consent = self.consent.to_dict()

        icon_url = self.icon_url

        live: dict[str, Any] | Unset = UNSET
        if not isinstance(self.live, Unset):
            live = self.live.to_dict()

        live_since: str | Unset = UNSET
        if not isinstance(self.live_since, Unset):
            live_since = self.live_since.isoformat()

        site: dict[str, Any] | Unset = UNSET
        if not isinstance(self.site, Unset):
            site = self.site.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canEdit": can_edit,
            "changed": changed,
            "draft": draft,
            "formId": form_id,
            "liveVersion": live_version,
            "name": name,
            "pushOn": push_on,
            "sites": sites,
            "stats": stats,
            "status": status,
            "updatedAt": updated_at,
            "updatedBy": updated_by,
        })
        if consent is not UNSET:
            field_dict["consent"] = consent
        if icon_url is not UNSET:
            field_dict["iconUrl"] = icon_url
        if live is not UNSET:
            field_dict["live"] = live
        if live_since is not UNSET:
            field_dict["liveSince"] = live_since
        if site is not UNSET:
            field_dict["site"] = site

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.push_consent_record import PushConsentRecord # noqa: PLC0415
        from ..models.push_prompt_config import PushPromptConfig # noqa: PLC0415
        from ..models.push_prompt_site_option import PushPromptSiteOption # noqa: PLC0415
        from ..models.push_prompt_stats import PushPromptStats # noqa: PLC0415
        d = dict(src_dict)
        can_edit = d.pop("canEdit")

        changed = d.pop("changed")

        draft = PushPromptConfig.from_dict(d.pop("draft"))




        form_id = d.pop("formId")

        live_version = d.pop("liveVersion")

        name = d.pop("name")

        push_on = d.pop("pushOn")

        sites = []
        _sites = d.pop("sites")
        for sites_item_data in (_sites):
            sites_item = PushPromptSiteOption.from_dict(sites_item_data)



            sites.append(sites_item)


        stats = PushPromptStats.from_dict(d.pop("stats"))




        status = PushPromptStatus(d.pop("status"))




        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        updated_by = d.pop("updatedBy")

        _consent = d.pop("consent", UNSET)
        consent: PushConsentRecord | Unset
        if isinstance(_consent,  Unset):
            consent = UNSET
        else:
            consent = PushConsentRecord.from_dict(_consent)




        icon_url = d.pop("iconUrl", UNSET)

        _live = d.pop("live", UNSET)
        live: PushPromptConfig | Unset
        if isinstance(_live,  Unset):
            live = UNSET
        else:
            live = PushPromptConfig.from_dict(_live)




        _live_since = d.pop("liveSince", UNSET)
        live_since: datetime.datetime | Unset
        if isinstance(_live_since,  Unset):
            live_since = UNSET
        else:
            live_since = datetime.datetime.fromisoformat(_live_since)




        _site = d.pop("site", UNSET)
        site: PushPromptSiteOption | Unset
        if isinstance(_site,  Unset):
            site = UNSET
        else:
            site = PushPromptSiteOption.from_dict(_site)




        push_prompt = cls(
            can_edit=can_edit,
            changed=changed,
            draft=draft,
            form_id=form_id,
            live_version=live_version,
            name=name,
            push_on=push_on,
            sites=sites,
            stats=stats,
            status=status,
            updated_at=updated_at,
            updated_by=updated_by,
            consent=consent,
            icon_url=icon_url,
            live=live,
            live_since=live_since,
            site=site,
        )

        return push_prompt

