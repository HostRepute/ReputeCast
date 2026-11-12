from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_campaign_status import PushCampaignStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.push_access import PushAccess
  from ..models.push_audience import PushAudience
  from ..models.push_audience_ref import PushAudienceRef
  from ..models.push_audience_site import PushAudienceSite
  from ..models.push_button import PushButton
  from ..models.push_guards import PushGuards
  from ..models.push_image import PushImage
  from ..models.push_length_limits import PushLengthLimits
  from ..models.push_schedule import PushSchedule
  from ..models.push_test_result import PushTestResult





T = TypeVar("T", bound="PushCampaign")



@_attrs_define
class PushCampaign:
    """ 
        Attributes:
            access (PushAccess):
            audience (PushAudience):
            buttons (list[PushButton]):
            created_by (str):
            guards (PushGuards):
            host (str):
            icon_url (str):
            id (str):
            limits (PushLengthLimits):
            link (str):
            lists (list[PushAudienceRef]):
            message (str):
            name (str):
            push_on (bool):
            replace_tag (bool):
            schedule (PushSchedule):
            segments (list[PushAudienceRef]):
            sites (list[PushAudienceSite]):
            status (PushCampaignStatus):
            tag (str):
            title (str):
            ttl_minutes (int):
            updated_at (datetime.datetime):
            updated_by (str):
            utm (str):
            version (int):
            autosaved_at (datetime.datetime | Unset):
            image (PushImage | Unset):
            test (PushTestResult | Unset):
     """

    access: PushAccess
    audience: PushAudience
    buttons: list[PushButton]
    created_by: str
    guards: PushGuards
    host: str
    icon_url: str
    id: str
    limits: PushLengthLimits
    link: str
    lists: list[PushAudienceRef]
    message: str
    name: str
    push_on: bool
    replace_tag: bool
    schedule: PushSchedule
    segments: list[PushAudienceRef]
    sites: list[PushAudienceSite]
    status: PushCampaignStatus
    tag: str
    title: str
    ttl_minutes: int
    updated_at: datetime.datetime
    updated_by: str
    utm: str
    version: int
    autosaved_at: datetime.datetime | Unset = UNSET
    image: PushImage | Unset = UNSET
    test: PushTestResult | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.push_access import PushAccess # noqa: PLC0415
        from ..models.push_audience import PushAudience # noqa: PLC0415
        from ..models.push_audience_ref import PushAudienceRef # noqa: PLC0415
        from ..models.push_audience_site import PushAudienceSite # noqa: PLC0415
        from ..models.push_button import PushButton # noqa: PLC0415
        from ..models.push_guards import PushGuards # noqa: PLC0415
        from ..models.push_image import PushImage # noqa: PLC0415
        from ..models.push_length_limits import PushLengthLimits # noqa: PLC0415
        from ..models.push_schedule import PushSchedule # noqa: PLC0415
        from ..models.push_test_result import PushTestResult # noqa: PLC0415
        access = self.access.to_dict()

        audience = self.audience.to_dict()

        buttons = []
        for buttons_item_data in self.buttons:
            buttons_item = buttons_item_data.to_dict()
            buttons.append(buttons_item)



        created_by = self.created_by

        guards = self.guards.to_dict()

        host = self.host

        icon_url = self.icon_url

        id = self.id

        limits = self.limits.to_dict()

        link = self.link

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        message = self.message

        name = self.name

        push_on = self.push_on

        replace_tag = self.replace_tag

        schedule = self.schedule.to_dict()

        segments = []
        for segments_item_data in self.segments:
            segments_item = segments_item_data.to_dict()
            segments.append(segments_item)



        sites = []
        for sites_item_data in self.sites:
            sites_item = sites_item_data.to_dict()
            sites.append(sites_item)



        status = self.status.value

        tag = self.tag

        title = self.title

        ttl_minutes = self.ttl_minutes

        updated_at = self.updated_at.isoformat()

        updated_by = self.updated_by

        utm = self.utm

        version = self.version

        autosaved_at: str | Unset = UNSET
        if not isinstance(self.autosaved_at, Unset):
            autosaved_at = self.autosaved_at.isoformat()

        image: dict[str, Any] | Unset = UNSET
        if not isinstance(self.image, Unset):
            image = self.image.to_dict()

        test: dict[str, Any] | Unset = UNSET
        if not isinstance(self.test, Unset):
            test = self.test.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "access": access,
            "audience": audience,
            "buttons": buttons,
            "createdBy": created_by,
            "guards": guards,
            "host": host,
            "iconUrl": icon_url,
            "id": id,
            "limits": limits,
            "link": link,
            "lists": lists,
            "message": message,
            "name": name,
            "pushOn": push_on,
            "replaceTag": replace_tag,
            "schedule": schedule,
            "segments": segments,
            "sites": sites,
            "status": status,
            "tag": tag,
            "title": title,
            "ttlMinutes": ttl_minutes,
            "updatedAt": updated_at,
            "updatedBy": updated_by,
            "utm": utm,
            "version": version,
        })
        if autosaved_at is not UNSET:
            field_dict["autosavedAt"] = autosaved_at
        if image is not UNSET:
            field_dict["image"] = image
        if test is not UNSET:
            field_dict["test"] = test

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.push_access import PushAccess # noqa: PLC0415
        from ..models.push_audience import PushAudience # noqa: PLC0415
        from ..models.push_audience_ref import PushAudienceRef # noqa: PLC0415
        from ..models.push_audience_site import PushAudienceSite # noqa: PLC0415
        from ..models.push_button import PushButton # noqa: PLC0415
        from ..models.push_guards import PushGuards # noqa: PLC0415
        from ..models.push_image import PushImage # noqa: PLC0415
        from ..models.push_length_limits import PushLengthLimits # noqa: PLC0415
        from ..models.push_schedule import PushSchedule # noqa: PLC0415
        from ..models.push_test_result import PushTestResult # noqa: PLC0415
        d = dict(src_dict)
        access = PushAccess.from_dict(d.pop("access"))




        audience = PushAudience.from_dict(d.pop("audience"))




        buttons = []
        _buttons = d.pop("buttons")
        for buttons_item_data in (_buttons):
            buttons_item = PushButton.from_dict(buttons_item_data)



            buttons.append(buttons_item)


        created_by = d.pop("createdBy")

        guards = PushGuards.from_dict(d.pop("guards"))




        host = d.pop("host")

        icon_url = d.pop("iconUrl")

        id = d.pop("id")

        limits = PushLengthLimits.from_dict(d.pop("limits"))




        link = d.pop("link")

        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = PushAudienceRef.from_dict(lists_item_data)



            lists.append(lists_item)


        message = d.pop("message")

        name = d.pop("name")

        push_on = d.pop("pushOn")

        replace_tag = d.pop("replaceTag")

        schedule = PushSchedule.from_dict(d.pop("schedule"))




        segments = []
        _segments = d.pop("segments")
        for segments_item_data in (_segments):
            segments_item = PushAudienceRef.from_dict(segments_item_data)



            segments.append(segments_item)


        sites = []
        _sites = d.pop("sites")
        for sites_item_data in (_sites):
            sites_item = PushAudienceSite.from_dict(sites_item_data)



            sites.append(sites_item)


        status = PushCampaignStatus(d.pop("status"))




        tag = d.pop("tag")

        title = d.pop("title")

        ttl_minutes = d.pop("ttlMinutes")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        updated_by = d.pop("updatedBy")

        utm = d.pop("utm")

        version = d.pop("version")

        _autosaved_at = d.pop("autosavedAt", UNSET)
        autosaved_at: datetime.datetime | Unset
        if isinstance(_autosaved_at,  Unset):
            autosaved_at = UNSET
        else:
            autosaved_at = datetime.datetime.fromisoformat(_autosaved_at)




        _image = d.pop("image", UNSET)
        image: PushImage | Unset
        if isinstance(_image,  Unset):
            image = UNSET
        else:
            image = PushImage.from_dict(_image)




        _test = d.pop("test", UNSET)
        test: PushTestResult | Unset
        if isinstance(_test,  Unset):
            test = UNSET
        else:
            test = PushTestResult.from_dict(_test)




        push_campaign = cls(
            access=access,
            audience=audience,
            buttons=buttons,
            created_by=created_by,
            guards=guards,
            host=host,
            icon_url=icon_url,
            id=id,
            limits=limits,
            link=link,
            lists=lists,
            message=message,
            name=name,
            push_on=push_on,
            replace_tag=replace_tag,
            schedule=schedule,
            segments=segments,
            sites=sites,
            status=status,
            tag=tag,
            title=title,
            ttl_minutes=ttl_minutes,
            updated_at=updated_at,
            updated_by=updated_by,
            utm=utm,
            version=version,
            autosaved_at=autosaved_at,
            image=image,
            test=test,
        )

        return push_campaign

