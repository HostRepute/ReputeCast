from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_profile_consent import ContactProfileConsent
from ..models.contact_profile_tier import ContactProfileTier
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.contact_language import ContactLanguage
  from ..models.contact_list_consent import ContactListConsent
  from ..models.contact_profile_field import ContactProfileField
  from ..models.contact_profile_preferences import ContactProfilePreferences
  from ..models.contact_profile_stats import ContactProfileStats
  from ..models.contact_profile_suppression import ContactProfileSuppression





T = TypeVar("T", bound="ContactProfile")



@_attrs_define
class ContactProfile:
    """ 
        Attributes:
            activity_total (int):
            consent (ContactProfileConsent):
            email (str):
            fields (list[ContactProfileField]):
            first_name (str):
            id (str):
            joined_at (datetime.datetime):
            language (ContactLanguage):
            last_name (str):
            lists (list[ContactListConsent]):
            mailbox_label (str):
            mailbox_provider (str):
            name (str):
            preferences (ContactProfilePreferences):
            role_address (bool):
            source (str):
            source_label (str):
            stats (ContactProfileStats):
            tags (list[str]):
            tier (ContactProfileTier):
            suppression (ContactProfileSuppression | Unset):
     """

    activity_total: int
    consent: ContactProfileConsent
    email: str
    fields: list[ContactProfileField]
    first_name: str
    id: str
    joined_at: datetime.datetime
    language: ContactLanguage
    last_name: str
    lists: list[ContactListConsent]
    mailbox_label: str
    mailbox_provider: str
    name: str
    preferences: ContactProfilePreferences
    role_address: bool
    source: str
    source_label: str
    stats: ContactProfileStats
    tags: list[str]
    tier: ContactProfileTier
    suppression: ContactProfileSuppression | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_language import ContactLanguage # noqa: PLC0415
        from ..models.contact_list_consent import ContactListConsent # noqa: PLC0415
        from ..models.contact_profile_field import ContactProfileField # noqa: PLC0415
        from ..models.contact_profile_preferences import ContactProfilePreferences # noqa: PLC0415
        from ..models.contact_profile_stats import ContactProfileStats # noqa: PLC0415
        from ..models.contact_profile_suppression import ContactProfileSuppression # noqa: PLC0415
        activity_total = self.activity_total

        consent = self.consent.value

        email = self.email

        fields = []
        for fields_item_data in self.fields:
            fields_item = fields_item_data.to_dict()
            fields.append(fields_item)



        first_name = self.first_name

        id = self.id

        joined_at = self.joined_at.isoformat()

        language = self.language.to_dict()

        last_name = self.last_name

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        mailbox_label = self.mailbox_label

        mailbox_provider = self.mailbox_provider

        name = self.name

        preferences = self.preferences.to_dict()

        role_address = self.role_address

        source = self.source

        source_label = self.source_label

        stats = self.stats.to_dict()

        tags = self.tags



        tier = self.tier.value

        suppression: dict[str, Any] | Unset = UNSET
        if not isinstance(self.suppression, Unset):
            suppression = self.suppression.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "activityTotal": activity_total,
            "consent": consent,
            "email": email,
            "fields": fields,
            "firstName": first_name,
            "id": id,
            "joinedAt": joined_at,
            "language": language,
            "lastName": last_name,
            "lists": lists,
            "mailboxLabel": mailbox_label,
            "mailboxProvider": mailbox_provider,
            "name": name,
            "preferences": preferences,
            "roleAddress": role_address,
            "source": source,
            "sourceLabel": source_label,
            "stats": stats,
            "tags": tags,
            "tier": tier,
        })
        if suppression is not UNSET:
            field_dict["suppression"] = suppression

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_language import ContactLanguage # noqa: PLC0415
        from ..models.contact_list_consent import ContactListConsent # noqa: PLC0415
        from ..models.contact_profile_field import ContactProfileField # noqa: PLC0415
        from ..models.contact_profile_preferences import ContactProfilePreferences # noqa: PLC0415
        from ..models.contact_profile_stats import ContactProfileStats # noqa: PLC0415
        from ..models.contact_profile_suppression import ContactProfileSuppression # noqa: PLC0415
        d = dict(src_dict)
        activity_total = d.pop("activityTotal")

        consent = ContactProfileConsent(d.pop("consent"))




        email = d.pop("email")

        fields = []
        _fields = d.pop("fields")
        for fields_item_data in (_fields):
            fields_item = ContactProfileField.from_dict(fields_item_data)



            fields.append(fields_item)


        first_name = d.pop("firstName")

        id = d.pop("id")

        joined_at = datetime.datetime.fromisoformat(d.pop("joinedAt"))




        language = ContactLanguage.from_dict(d.pop("language"))




        last_name = d.pop("lastName")

        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = ContactListConsent.from_dict(lists_item_data)



            lists.append(lists_item)


        mailbox_label = d.pop("mailboxLabel")

        mailbox_provider = d.pop("mailboxProvider")

        name = d.pop("name")

        preferences = ContactProfilePreferences.from_dict(d.pop("preferences"))




        role_address = d.pop("roleAddress")

        source = d.pop("source")

        source_label = d.pop("sourceLabel")

        stats = ContactProfileStats.from_dict(d.pop("stats"))




        tags = cast(list[str], d.pop("tags"))


        tier = ContactProfileTier(d.pop("tier"))




        _suppression = d.pop("suppression", UNSET)
        suppression: ContactProfileSuppression | Unset
        if isinstance(_suppression,  Unset):
            suppression = UNSET
        else:
            suppression = ContactProfileSuppression.from_dict(_suppression)




        contact_profile = cls(
            activity_total=activity_total,
            consent=consent,
            email=email,
            fields=fields,
            first_name=first_name,
            id=id,
            joined_at=joined_at,
            language=language,
            last_name=last_name,
            lists=lists,
            mailbox_label=mailbox_label,
            mailbox_provider=mailbox_provider,
            name=name,
            preferences=preferences,
            role_address=role_address,
            source=source,
            source_label=source_label,
            stats=stats,
            tags=tags,
            tier=tier,
            suppression=suppression,
        )

        return contact_profile

