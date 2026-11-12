from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_template_editor_mode import TransactionalTemplateEditorMode
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.transactional_category_ref import TransactionalCategoryRef
  from ..models.transactional_sender_ref import TransactionalSenderRef
  from ..models.transactional_template_access import TransactionalTemplateAccess
  from ..models.transactional_template_origin import TransactionalTemplateOrigin
  from ..models.transactional_test_data import TransactionalTestData
  from ..models.transactional_variable import TransactionalVariable





T = TypeVar("T", bound="TransactionalTemplate")



@_attrs_define
class TransactionalTemplate:
    """ 
        Attributes:
            access (TransactionalTemplateAccess):
            archived (bool):
            created_at (datetime.datetime):
            description (str):
            draft_changes (bool):
            draft_hash (str):
            draft_revision (int):
            draft_saved_at (datetime.datetime):
            draft_saved_by (str):
            editor_mode (TransactionalTemplateEditorMode):
            html (str):
            id (str):
            key (str): The ID the API sends, fixed after the first publish
            key_locked (bool):
            language (str):
            mjml (str):
            name (str):
            origin (TransactionalTemplateOrigin):
            plain_text (str):
            preheader (str):
            reply_to (str):
            source_language (str):
            subject (str):
            test_data (list[TransactionalTestData]):
            track_clicks (bool):
            track_opens (bool):
            updated_at (datetime.datetime):
            variables (list[TransactionalVariable]):
            versions_count (int):
            archived_at (datetime.datetime | Unset):
            category (TransactionalCategoryRef | Unset):
            live_by (str | Unset):
            live_since (datetime.datetime | Unset):
            live_version (int | Unset):
            sender (TransactionalSenderRef | Unset):
     """

    access: TransactionalTemplateAccess
    archived: bool
    created_at: datetime.datetime
    description: str
    draft_changes: bool
    draft_hash: str
    draft_revision: int
    draft_saved_at: datetime.datetime
    draft_saved_by: str
    editor_mode: TransactionalTemplateEditorMode
    html: str
    id: str
    key: str
    key_locked: bool
    language: str
    mjml: str
    name: str
    origin: TransactionalTemplateOrigin
    plain_text: str
    preheader: str
    reply_to: str
    source_language: str
    subject: str
    test_data: list[TransactionalTestData]
    track_clicks: bool
    track_opens: bool
    updated_at: datetime.datetime
    variables: list[TransactionalVariable]
    versions_count: int
    archived_at: datetime.datetime | Unset = UNSET
    category: TransactionalCategoryRef | Unset = UNSET
    live_by: str | Unset = UNSET
    live_since: datetime.datetime | Unset = UNSET
    live_version: int | Unset = UNSET
    sender: TransactionalSenderRef | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_category_ref import TransactionalCategoryRef # noqa: PLC0415
        from ..models.transactional_sender_ref import TransactionalSenderRef # noqa: PLC0415
        from ..models.transactional_template_access import TransactionalTemplateAccess # noqa: PLC0415
        from ..models.transactional_template_origin import TransactionalTemplateOrigin # noqa: PLC0415
        from ..models.transactional_test_data import TransactionalTestData # noqa: PLC0415
        from ..models.transactional_variable import TransactionalVariable # noqa: PLC0415
        access = self.access.to_dict()

        archived = self.archived

        created_at = self.created_at.isoformat()

        description = self.description

        draft_changes = self.draft_changes

        draft_hash = self.draft_hash

        draft_revision = self.draft_revision

        draft_saved_at = self.draft_saved_at.isoformat()

        draft_saved_by = self.draft_saved_by

        editor_mode = self.editor_mode.value

        html = self.html

        id = self.id

        key = self.key

        key_locked = self.key_locked

        language = self.language

        mjml = self.mjml

        name = self.name

        origin = self.origin.to_dict()

        plain_text = self.plain_text

        preheader = self.preheader

        reply_to = self.reply_to

        source_language = self.source_language

        subject = self.subject

        test_data = []
        for test_data_item_data in self.test_data:
            test_data_item = test_data_item_data.to_dict()
            test_data.append(test_data_item)



        track_clicks = self.track_clicks

        track_opens = self.track_opens

        updated_at = self.updated_at.isoformat()

        variables = []
        for variables_item_data in self.variables:
            variables_item = variables_item_data.to_dict()
            variables.append(variables_item)



        versions_count = self.versions_count

        archived_at: str | Unset = UNSET
        if not isinstance(self.archived_at, Unset):
            archived_at = self.archived_at.isoformat()

        category: dict[str, Any] | Unset = UNSET
        if not isinstance(self.category, Unset):
            category = self.category.to_dict()

        live_by = self.live_by

        live_since: str | Unset = UNSET
        if not isinstance(self.live_since, Unset):
            live_since = self.live_since.isoformat()

        live_version = self.live_version

        sender: dict[str, Any] | Unset = UNSET
        if not isinstance(self.sender, Unset):
            sender = self.sender.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "access": access,
            "archived": archived,
            "createdAt": created_at,
            "description": description,
            "draftChanges": draft_changes,
            "draftHash": draft_hash,
            "draftRevision": draft_revision,
            "draftSavedAt": draft_saved_at,
            "draftSavedBy": draft_saved_by,
            "editorMode": editor_mode,
            "html": html,
            "id": id,
            "key": key,
            "keyLocked": key_locked,
            "language": language,
            "mjml": mjml,
            "name": name,
            "origin": origin,
            "plainText": plain_text,
            "preheader": preheader,
            "replyTo": reply_to,
            "sourceLanguage": source_language,
            "subject": subject,
            "testData": test_data,
            "trackClicks": track_clicks,
            "trackOpens": track_opens,
            "updatedAt": updated_at,
            "variables": variables,
            "versionsCount": versions_count,
        })
        if archived_at is not UNSET:
            field_dict["archivedAt"] = archived_at
        if category is not UNSET:
            field_dict["category"] = category
        if live_by is not UNSET:
            field_dict["liveBy"] = live_by
        if live_since is not UNSET:
            field_dict["liveSince"] = live_since
        if live_version is not UNSET:
            field_dict["liveVersion"] = live_version
        if sender is not UNSET:
            field_dict["sender"] = sender

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_category_ref import TransactionalCategoryRef # noqa: PLC0415
        from ..models.transactional_sender_ref import TransactionalSenderRef # noqa: PLC0415
        from ..models.transactional_template_access import TransactionalTemplateAccess # noqa: PLC0415
        from ..models.transactional_template_origin import TransactionalTemplateOrigin # noqa: PLC0415
        from ..models.transactional_test_data import TransactionalTestData # noqa: PLC0415
        from ..models.transactional_variable import TransactionalVariable # noqa: PLC0415
        d = dict(src_dict)
        access = TransactionalTemplateAccess.from_dict(d.pop("access"))




        archived = d.pop("archived")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        description = d.pop("description")

        draft_changes = d.pop("draftChanges")

        draft_hash = d.pop("draftHash")

        draft_revision = d.pop("draftRevision")

        draft_saved_at = datetime.datetime.fromisoformat(d.pop("draftSavedAt"))




        draft_saved_by = d.pop("draftSavedBy")

        editor_mode = TransactionalTemplateEditorMode(d.pop("editorMode"))




        html = d.pop("html")

        id = d.pop("id")

        key = d.pop("key")

        key_locked = d.pop("keyLocked")

        language = d.pop("language")

        mjml = d.pop("mjml")

        name = d.pop("name")

        origin = TransactionalTemplateOrigin.from_dict(d.pop("origin"))




        plain_text = d.pop("plainText")

        preheader = d.pop("preheader")

        reply_to = d.pop("replyTo")

        source_language = d.pop("sourceLanguage")

        subject = d.pop("subject")

        test_data = []
        _test_data = d.pop("testData")
        for test_data_item_data in (_test_data):
            test_data_item = TransactionalTestData.from_dict(test_data_item_data)



            test_data.append(test_data_item)


        track_clicks = d.pop("trackClicks")

        track_opens = d.pop("trackOpens")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        variables = []
        _variables = d.pop("variables")
        for variables_item_data in (_variables):
            variables_item = TransactionalVariable.from_dict(variables_item_data)



            variables.append(variables_item)


        versions_count = d.pop("versionsCount")

        _archived_at = d.pop("archivedAt", UNSET)
        archived_at: datetime.datetime | Unset
        if isinstance(_archived_at,  Unset):
            archived_at = UNSET
        else:
            archived_at = datetime.datetime.fromisoformat(_archived_at)




        _category = d.pop("category", UNSET)
        category: TransactionalCategoryRef | Unset
        if isinstance(_category,  Unset):
            category = UNSET
        else:
            category = TransactionalCategoryRef.from_dict(_category)




        live_by = d.pop("liveBy", UNSET)

        _live_since = d.pop("liveSince", UNSET)
        live_since: datetime.datetime | Unset
        if isinstance(_live_since,  Unset):
            live_since = UNSET
        else:
            live_since = datetime.datetime.fromisoformat(_live_since)




        live_version = d.pop("liveVersion", UNSET)

        _sender = d.pop("sender", UNSET)
        sender: TransactionalSenderRef | Unset
        if isinstance(_sender,  Unset):
            sender = UNSET
        else:
            sender = TransactionalSenderRef.from_dict(_sender)




        transactional_template = cls(
            access=access,
            archived=archived,
            created_at=created_at,
            description=description,
            draft_changes=draft_changes,
            draft_hash=draft_hash,
            draft_revision=draft_revision,
            draft_saved_at=draft_saved_at,
            draft_saved_by=draft_saved_by,
            editor_mode=editor_mode,
            html=html,
            id=id,
            key=key,
            key_locked=key_locked,
            language=language,
            mjml=mjml,
            name=name,
            origin=origin,
            plain_text=plain_text,
            preheader=preheader,
            reply_to=reply_to,
            source_language=source_language,
            subject=subject,
            test_data=test_data,
            track_clicks=track_clicks,
            track_opens=track_opens,
            updated_at=updated_at,
            variables=variables,
            versions_count=versions_count,
            archived_at=archived_at,
            category=category,
            live_by=live_by,
            live_since=live_since,
            live_version=live_version,
            sender=sender,
        )

        return transactional_template

