from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.composer_campaign_digest_role import ComposerCampaignDigestRole
from ..models.composer_campaign_editor_mode import ComposerCampaignEditorMode
from ..models.composer_campaign_status import ComposerCampaignStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.composer_ab_test import ComposerABTest
  from ..models.composer_access import ComposerAccess
  from ..models.composer_approval import ComposerApproval
  from ..models.composer_audience import ComposerAudience
  from ..models.composer_content_settings import ComposerContentSettings
  from ..models.composer_route import ComposerRoute
  from ..models.composer_sender_ref import ComposerSenderRef
  from ..models.readiness_summary import ReadinessSummary





T = TypeVar("T", bound="ComposerCampaign")



@_attrs_define
class ComposerCampaign:
    """ 
        Attributes:
            access (ComposerAccess):
            audience (ComposerAudience):
            audience_label (str):
            content_hash (str):
            created_at (datetime.datetime):
            editor_mode (ComposerCampaignEditorMode):
            html (str):
            id (str):
            language (str):
            mjml (str):
            name (str):
            plain_text (str):
            preheader (str):
            reply_to (str):
            route (ComposerRoute):
            settings (ComposerContentSettings):
            source_language (str):
            status (ComposerCampaignStatus):
            subject (str):
            updated_at (datetime.datetime):
            updated_by (str):
            version (int):
            version_count (int):
            ab_test (ComposerABTest | Unset):
            approval (ComposerApproval | Unset):
            audience_count (int | Unset):
            autosaved_at (datetime.datetime | Unset):
            digest_id (str | Unset):
            digest_role (ComposerCampaignDigestRole | Unset):
            readiness (ReadinessSummary | Unset):
            scheduled_at (datetime.datetime | Unset):
            sender (ComposerSenderRef | Unset):
     """

    access: ComposerAccess
    audience: ComposerAudience
    audience_label: str
    content_hash: str
    created_at: datetime.datetime
    editor_mode: ComposerCampaignEditorMode
    html: str
    id: str
    language: str
    mjml: str
    name: str
    plain_text: str
    preheader: str
    reply_to: str
    route: ComposerRoute
    settings: ComposerContentSettings
    source_language: str
    status: ComposerCampaignStatus
    subject: str
    updated_at: datetime.datetime
    updated_by: str
    version: int
    version_count: int
    ab_test: ComposerABTest | Unset = UNSET
    approval: ComposerApproval | Unset = UNSET
    audience_count: int | Unset = UNSET
    autosaved_at: datetime.datetime | Unset = UNSET
    digest_id: str | Unset = UNSET
    digest_role: ComposerCampaignDigestRole | Unset = UNSET
    readiness: ReadinessSummary | Unset = UNSET
    scheduled_at: datetime.datetime | Unset = UNSET
    sender: ComposerSenderRef | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.composer_ab_test import ComposerABTest # noqa: PLC0415
        from ..models.composer_access import ComposerAccess # noqa: PLC0415
        from ..models.composer_approval import ComposerApproval # noqa: PLC0415
        from ..models.composer_audience import ComposerAudience # noqa: PLC0415
        from ..models.composer_content_settings import ComposerContentSettings # noqa: PLC0415
        from ..models.composer_route import ComposerRoute # noqa: PLC0415
        from ..models.composer_sender_ref import ComposerSenderRef # noqa: PLC0415
        from ..models.readiness_summary import ReadinessSummary # noqa: PLC0415
        access = self.access.to_dict()

        audience = self.audience.to_dict()

        audience_label = self.audience_label

        content_hash = self.content_hash

        created_at = self.created_at.isoformat()

        editor_mode = self.editor_mode.value

        html = self.html

        id = self.id

        language = self.language

        mjml = self.mjml

        name = self.name

        plain_text = self.plain_text

        preheader = self.preheader

        reply_to = self.reply_to

        route = self.route.to_dict()

        settings = self.settings.to_dict()

        source_language = self.source_language

        status = self.status.value

        subject = self.subject

        updated_at = self.updated_at.isoformat()

        updated_by = self.updated_by

        version = self.version

        version_count = self.version_count

        ab_test: dict[str, Any] | Unset = UNSET
        if not isinstance(self.ab_test, Unset):
            ab_test = self.ab_test.to_dict()

        approval: dict[str, Any] | Unset = UNSET
        if not isinstance(self.approval, Unset):
            approval = self.approval.to_dict()

        audience_count = self.audience_count

        autosaved_at: str | Unset = UNSET
        if not isinstance(self.autosaved_at, Unset):
            autosaved_at = self.autosaved_at.isoformat()

        digest_id = self.digest_id

        digest_role: str | Unset = UNSET
        if not isinstance(self.digest_role, Unset):
            digest_role = self.digest_role.value


        readiness: dict[str, Any] | Unset = UNSET
        if not isinstance(self.readiness, Unset):
            readiness = self.readiness.to_dict()

        scheduled_at: str | Unset = UNSET
        if not isinstance(self.scheduled_at, Unset):
            scheduled_at = self.scheduled_at.isoformat()

        sender: dict[str, Any] | Unset = UNSET
        if not isinstance(self.sender, Unset):
            sender = self.sender.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "access": access,
            "audience": audience,
            "audienceLabel": audience_label,
            "contentHash": content_hash,
            "createdAt": created_at,
            "editorMode": editor_mode,
            "html": html,
            "id": id,
            "language": language,
            "mjml": mjml,
            "name": name,
            "plainText": plain_text,
            "preheader": preheader,
            "replyTo": reply_to,
            "route": route,
            "settings": settings,
            "sourceLanguage": source_language,
            "status": status,
            "subject": subject,
            "updatedAt": updated_at,
            "updatedBy": updated_by,
            "version": version,
            "versionCount": version_count,
        })
        if ab_test is not UNSET:
            field_dict["abTest"] = ab_test
        if approval is not UNSET:
            field_dict["approval"] = approval
        if audience_count is not UNSET:
            field_dict["audienceCount"] = audience_count
        if autosaved_at is not UNSET:
            field_dict["autosavedAt"] = autosaved_at
        if digest_id is not UNSET:
            field_dict["digestId"] = digest_id
        if digest_role is not UNSET:
            field_dict["digestRole"] = digest_role
        if readiness is not UNSET:
            field_dict["readiness"] = readiness
        if scheduled_at is not UNSET:
            field_dict["scheduledAt"] = scheduled_at
        if sender is not UNSET:
            field_dict["sender"] = sender

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.composer_ab_test import ComposerABTest # noqa: PLC0415
        from ..models.composer_access import ComposerAccess # noqa: PLC0415
        from ..models.composer_approval import ComposerApproval # noqa: PLC0415
        from ..models.composer_audience import ComposerAudience # noqa: PLC0415
        from ..models.composer_content_settings import ComposerContentSettings # noqa: PLC0415
        from ..models.composer_route import ComposerRoute # noqa: PLC0415
        from ..models.composer_sender_ref import ComposerSenderRef # noqa: PLC0415
        from ..models.readiness_summary import ReadinessSummary # noqa: PLC0415
        d = dict(src_dict)
        access = ComposerAccess.from_dict(d.pop("access"))




        audience = ComposerAudience.from_dict(d.pop("audience"))




        audience_label = d.pop("audienceLabel")

        content_hash = d.pop("contentHash")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        editor_mode = ComposerCampaignEditorMode(d.pop("editorMode"))




        html = d.pop("html")

        id = d.pop("id")

        language = d.pop("language")

        mjml = d.pop("mjml")

        name = d.pop("name")

        plain_text = d.pop("plainText")

        preheader = d.pop("preheader")

        reply_to = d.pop("replyTo")

        route = ComposerRoute.from_dict(d.pop("route"))




        settings = ComposerContentSettings.from_dict(d.pop("settings"))




        source_language = d.pop("sourceLanguage")

        status = ComposerCampaignStatus(d.pop("status"))




        subject = d.pop("subject")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        updated_by = d.pop("updatedBy")

        version = d.pop("version")

        version_count = d.pop("versionCount")

        _ab_test = d.pop("abTest", UNSET)
        ab_test: ComposerABTest | Unset
        if isinstance(_ab_test,  Unset):
            ab_test = UNSET
        else:
            ab_test = ComposerABTest.from_dict(_ab_test)




        _approval = d.pop("approval", UNSET)
        approval: ComposerApproval | Unset
        if isinstance(_approval,  Unset):
            approval = UNSET
        else:
            approval = ComposerApproval.from_dict(_approval)




        audience_count = d.pop("audienceCount", UNSET)

        _autosaved_at = d.pop("autosavedAt", UNSET)
        autosaved_at: datetime.datetime | Unset
        if isinstance(_autosaved_at,  Unset):
            autosaved_at = UNSET
        else:
            autosaved_at = datetime.datetime.fromisoformat(_autosaved_at)




        digest_id = d.pop("digestId", UNSET)

        _digest_role = d.pop("digestRole", UNSET)
        digest_role: ComposerCampaignDigestRole | Unset
        if isinstance(_digest_role,  Unset):
            digest_role = UNSET
        else:
            digest_role = ComposerCampaignDigestRole(_digest_role)




        _readiness = d.pop("readiness", UNSET)
        readiness: ReadinessSummary | Unset
        if isinstance(_readiness,  Unset):
            readiness = UNSET
        else:
            readiness = ReadinessSummary.from_dict(_readiness)




        _scheduled_at = d.pop("scheduledAt", UNSET)
        scheduled_at: datetime.datetime | Unset
        if isinstance(_scheduled_at,  Unset):
            scheduled_at = UNSET
        else:
            scheduled_at = datetime.datetime.fromisoformat(_scheduled_at)




        _sender = d.pop("sender", UNSET)
        sender: ComposerSenderRef | Unset
        if isinstance(_sender,  Unset):
            sender = UNSET
        else:
            sender = ComposerSenderRef.from_dict(_sender)




        composer_campaign = cls(
            access=access,
            audience=audience,
            audience_label=audience_label,
            content_hash=content_hash,
            created_at=created_at,
            editor_mode=editor_mode,
            html=html,
            id=id,
            language=language,
            mjml=mjml,
            name=name,
            plain_text=plain_text,
            preheader=preheader,
            reply_to=reply_to,
            route=route,
            settings=settings,
            source_language=source_language,
            status=status,
            subject=subject,
            updated_at=updated_at,
            updated_by=updated_by,
            version=version,
            version_count=version_count,
            ab_test=ab_test,
            approval=approval,
            audience_count=audience_count,
            autosaved_at=autosaved_at,
            digest_id=digest_id,
            digest_role=digest_role,
            readiness=readiness,
            scheduled_at=scheduled_at,
            sender=sender,
        )

        return composer_campaign

