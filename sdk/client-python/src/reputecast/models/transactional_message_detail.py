from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_message_detail_status import TransactionalMessageDetailStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.transactional_attachment_meta import TransactionalAttachmentMeta
  from ..models.transactional_attempt import TransactionalAttempt
  from ..models.transactional_bounce import TransactionalBounce
  from ..models.transactional_content_state import TransactionalContentState
  from ..models.transactional_failover import TransactionalFailover
  from ..models.transactional_log_template import TransactionalLogTemplate
  from ..models.transactional_message_detail_metadata import TransactionalMessageDetailMetadata
  from ..models.transactional_message_language import TransactionalMessageLanguage
  from ..models.transactional_ref import TransactionalRef
  from ..models.transactional_rejection import TransactionalRejection
  from ..models.transactional_request import TransactionalRequest
  from ..models.transactional_resend_state import TransactionalResendState
  from ..models.transactional_route_info import TransactionalRouteInfo
  from ..models.transactional_suppression_info import TransactionalSuppressionInfo
  from ..models.transactional_timeline_entry import TransactionalTimelineEntry
  from ..models.transactional_webhook_delivery import TransactionalWebhookDelivery





T = TypeVar("T", bound="TransactionalMessageDetail")



@_attrs_define
class TransactionalMessageDetail:
    """ 
        Attributes:
            at (datetime.datetime): When the call was accepted
            attachments (list[TransactionalAttachmentMeta]):
            attempts (list[TransactionalAttempt]):
            bcc (list[str]):
            can_remove_suppression (bool):
            cc (list[str]):
            content (TransactionalContentState):
            content_bytes (int):
            deferrals (int):
            dropped (list[str]):
            email (str):
            from_ (str):
            id (str):
            metadata (TransactionalMessageDetailMetadata):
            raw (bool):
            request (TransactionalRequest):
            resend (TransactionalResendState):
            role_address (bool):
            route (TransactionalRouteInfo):
            sensitive_sent (list[str]):
            size_bytes (int):
            soft_bounce_limit (int):
            source (str):
            status (TransactionalMessageDetailStatus):
            subject (str):
            tags (list[str]):
            test_delivered (bool):
            timeline (list[TransactionalTimelineEntry]):
            tracked (bool):
            variables (int):
            webhooks (list[TransactionalWebhookDelivery]):
            bounce (TransactionalBounce | Unset):
            category (str | Unset):
            category_kind (str | Unset):
            clicked_at (datetime.datetime | Unset):
            contact (TransactionalRef | Unset):
            contact_id (str | Unset):
            delivery_ms (int | Unset):
            error (str | Unset):
            failed_over (TransactionalFailover | Unset):
            idempotency_key (str | Unset):
            ip (str | Unset):
            key (str | Unset):
            key_ref (TransactionalRef | Unset):
            language_info (TransactionalMessageLanguage | Unset):
            name (str | Unset):
            opened_at (datetime.datetime | Unset):
            provider (TransactionalRef | Unset):
            provider_message_id (str | Unset):
            rejection (TransactionalRejection | Unset):
            reply_to (str | Unset):
            resent_from (str | Unset):
            send_at (datetime.datetime | Unset):
            sent_by (str | Unset):
            soft_bounces (int | Unset):
            suppression (TransactionalSuppressionInfo | Unset):
            template (TransactionalLogTemplate | Unset):
     """

    at: datetime.datetime
    attachments: list[TransactionalAttachmentMeta]
    attempts: list[TransactionalAttempt]
    bcc: list[str]
    can_remove_suppression: bool
    cc: list[str]
    content: TransactionalContentState
    content_bytes: int
    deferrals: int
    dropped: list[str]
    email: str
    from_: str
    id: str
    metadata: TransactionalMessageDetailMetadata
    raw: bool
    request: TransactionalRequest
    resend: TransactionalResendState
    role_address: bool
    route: TransactionalRouteInfo
    sensitive_sent: list[str]
    size_bytes: int
    soft_bounce_limit: int
    source: str
    status: TransactionalMessageDetailStatus
    subject: str
    tags: list[str]
    test_delivered: bool
    timeline: list[TransactionalTimelineEntry]
    tracked: bool
    variables: int
    webhooks: list[TransactionalWebhookDelivery]
    bounce: TransactionalBounce | Unset = UNSET
    category: str | Unset = UNSET
    category_kind: str | Unset = UNSET
    clicked_at: datetime.datetime | Unset = UNSET
    contact: TransactionalRef | Unset = UNSET
    contact_id: str | Unset = UNSET
    delivery_ms: int | Unset = UNSET
    error: str | Unset = UNSET
    failed_over: TransactionalFailover | Unset = UNSET
    idempotency_key: str | Unset = UNSET
    ip: str | Unset = UNSET
    key: str | Unset = UNSET
    key_ref: TransactionalRef | Unset = UNSET
    language_info: TransactionalMessageLanguage | Unset = UNSET
    name: str | Unset = UNSET
    opened_at: datetime.datetime | Unset = UNSET
    provider: TransactionalRef | Unset = UNSET
    provider_message_id: str | Unset = UNSET
    rejection: TransactionalRejection | Unset = UNSET
    reply_to: str | Unset = UNSET
    resent_from: str | Unset = UNSET
    send_at: datetime.datetime | Unset = UNSET
    sent_by: str | Unset = UNSET
    soft_bounces: int | Unset = UNSET
    suppression: TransactionalSuppressionInfo | Unset = UNSET
    template: TransactionalLogTemplate | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_attachment_meta import TransactionalAttachmentMeta # noqa: PLC0415
        from ..models.transactional_attempt import TransactionalAttempt # noqa: PLC0415
        from ..models.transactional_bounce import TransactionalBounce # noqa: PLC0415
        from ..models.transactional_content_state import TransactionalContentState # noqa: PLC0415
        from ..models.transactional_failover import TransactionalFailover # noqa: PLC0415
        from ..models.transactional_log_template import TransactionalLogTemplate # noqa: PLC0415
        from ..models.transactional_message_detail_metadata import TransactionalMessageDetailMetadata # noqa: PLC0415
        from ..models.transactional_message_language import TransactionalMessageLanguage # noqa: PLC0415
        from ..models.transactional_ref import TransactionalRef # noqa: PLC0415
        from ..models.transactional_rejection import TransactionalRejection # noqa: PLC0415
        from ..models.transactional_request import TransactionalRequest # noqa: PLC0415
        from ..models.transactional_resend_state import TransactionalResendState # noqa: PLC0415
        from ..models.transactional_route_info import TransactionalRouteInfo # noqa: PLC0415
        from ..models.transactional_suppression_info import TransactionalSuppressionInfo # noqa: PLC0415
        from ..models.transactional_timeline_entry import TransactionalTimelineEntry # noqa: PLC0415
        from ..models.transactional_webhook_delivery import TransactionalWebhookDelivery # noqa: PLC0415
        at = self.at.isoformat()

        attachments = []
        for attachments_item_data in self.attachments:
            attachments_item = attachments_item_data.to_dict()
            attachments.append(attachments_item)



        attempts = []
        for attempts_item_data in self.attempts:
            attempts_item = attempts_item_data.to_dict()
            attempts.append(attempts_item)



        bcc = self.bcc



        can_remove_suppression = self.can_remove_suppression

        cc = self.cc



        content = self.content.to_dict()

        content_bytes = self.content_bytes

        deferrals = self.deferrals

        dropped = self.dropped



        email = self.email

        from_ = self.from_

        id = self.id

        metadata = self.metadata.to_dict()

        raw = self.raw

        request = self.request.to_dict()

        resend = self.resend.to_dict()

        role_address = self.role_address

        route = self.route.to_dict()

        sensitive_sent = self.sensitive_sent



        size_bytes = self.size_bytes

        soft_bounce_limit = self.soft_bounce_limit

        source = self.source

        status = self.status.value

        subject = self.subject

        tags = self.tags



        test_delivered = self.test_delivered

        timeline = []
        for timeline_item_data in self.timeline:
            timeline_item = timeline_item_data.to_dict()
            timeline.append(timeline_item)



        tracked = self.tracked

        variables = self.variables

        webhooks = []
        for webhooks_item_data in self.webhooks:
            webhooks_item = webhooks_item_data.to_dict()
            webhooks.append(webhooks_item)



        bounce: dict[str, Any] | Unset = UNSET
        if not isinstance(self.bounce, Unset):
            bounce = self.bounce.to_dict()

        category = self.category

        category_kind = self.category_kind

        clicked_at: str | Unset = UNSET
        if not isinstance(self.clicked_at, Unset):
            clicked_at = self.clicked_at.isoformat()

        contact: dict[str, Any] | Unset = UNSET
        if not isinstance(self.contact, Unset):
            contact = self.contact.to_dict()

        contact_id = self.contact_id

        delivery_ms = self.delivery_ms

        error = self.error

        failed_over: dict[str, Any] | Unset = UNSET
        if not isinstance(self.failed_over, Unset):
            failed_over = self.failed_over.to_dict()

        idempotency_key = self.idempotency_key

        ip = self.ip

        key = self.key

        key_ref: dict[str, Any] | Unset = UNSET
        if not isinstance(self.key_ref, Unset):
            key_ref = self.key_ref.to_dict()

        language_info: dict[str, Any] | Unset = UNSET
        if not isinstance(self.language_info, Unset):
            language_info = self.language_info.to_dict()

        name = self.name

        opened_at: str | Unset = UNSET
        if not isinstance(self.opened_at, Unset):
            opened_at = self.opened_at.isoformat()

        provider: dict[str, Any] | Unset = UNSET
        if not isinstance(self.provider, Unset):
            provider = self.provider.to_dict()

        provider_message_id = self.provider_message_id

        rejection: dict[str, Any] | Unset = UNSET
        if not isinstance(self.rejection, Unset):
            rejection = self.rejection.to_dict()

        reply_to = self.reply_to

        resent_from = self.resent_from

        send_at: str | Unset = UNSET
        if not isinstance(self.send_at, Unset):
            send_at = self.send_at.isoformat()

        sent_by = self.sent_by

        soft_bounces = self.soft_bounces

        suppression: dict[str, Any] | Unset = UNSET
        if not isinstance(self.suppression, Unset):
            suppression = self.suppression.to_dict()

        template: dict[str, Any] | Unset = UNSET
        if not isinstance(self.template, Unset):
            template = self.template.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "attachments": attachments,
            "attempts": attempts,
            "bcc": bcc,
            "canRemoveSuppression": can_remove_suppression,
            "cc": cc,
            "content": content,
            "contentBytes": content_bytes,
            "deferrals": deferrals,
            "dropped": dropped,
            "email": email,
            "from": from_,
            "id": id,
            "metadata": metadata,
            "raw": raw,
            "request": request,
            "resend": resend,
            "roleAddress": role_address,
            "route": route,
            "sensitiveSent": sensitive_sent,
            "sizeBytes": size_bytes,
            "softBounceLimit": soft_bounce_limit,
            "source": source,
            "status": status,
            "subject": subject,
            "tags": tags,
            "testDelivered": test_delivered,
            "timeline": timeline,
            "tracked": tracked,
            "variables": variables,
            "webhooks": webhooks,
        })
        if bounce is not UNSET:
            field_dict["bounce"] = bounce
        if category is not UNSET:
            field_dict["category"] = category
        if category_kind is not UNSET:
            field_dict["categoryKind"] = category_kind
        if clicked_at is not UNSET:
            field_dict["clickedAt"] = clicked_at
        if contact is not UNSET:
            field_dict["contact"] = contact
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id
        if delivery_ms is not UNSET:
            field_dict["deliveryMs"] = delivery_ms
        if error is not UNSET:
            field_dict["error"] = error
        if failed_over is not UNSET:
            field_dict["failedOver"] = failed_over
        if idempotency_key is not UNSET:
            field_dict["idempotencyKey"] = idempotency_key
        if ip is not UNSET:
            field_dict["ip"] = ip
        if key is not UNSET:
            field_dict["key"] = key
        if key_ref is not UNSET:
            field_dict["keyRef"] = key_ref
        if language_info is not UNSET:
            field_dict["languageInfo"] = language_info
        if name is not UNSET:
            field_dict["name"] = name
        if opened_at is not UNSET:
            field_dict["openedAt"] = opened_at
        if provider is not UNSET:
            field_dict["provider"] = provider
        if provider_message_id is not UNSET:
            field_dict["providerMessageId"] = provider_message_id
        if rejection is not UNSET:
            field_dict["rejection"] = rejection
        if reply_to is not UNSET:
            field_dict["replyTo"] = reply_to
        if resent_from is not UNSET:
            field_dict["resentFrom"] = resent_from
        if send_at is not UNSET:
            field_dict["sendAt"] = send_at
        if sent_by is not UNSET:
            field_dict["sentBy"] = sent_by
        if soft_bounces is not UNSET:
            field_dict["softBounces"] = soft_bounces
        if suppression is not UNSET:
            field_dict["suppression"] = suppression
        if template is not UNSET:
            field_dict["template"] = template

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_attachment_meta import TransactionalAttachmentMeta # noqa: PLC0415
        from ..models.transactional_attempt import TransactionalAttempt # noqa: PLC0415
        from ..models.transactional_bounce import TransactionalBounce # noqa: PLC0415
        from ..models.transactional_content_state import TransactionalContentState # noqa: PLC0415
        from ..models.transactional_failover import TransactionalFailover # noqa: PLC0415
        from ..models.transactional_log_template import TransactionalLogTemplate # noqa: PLC0415
        from ..models.transactional_message_detail_metadata import TransactionalMessageDetailMetadata # noqa: PLC0415
        from ..models.transactional_message_language import TransactionalMessageLanguage # noqa: PLC0415
        from ..models.transactional_ref import TransactionalRef # noqa: PLC0415
        from ..models.transactional_rejection import TransactionalRejection # noqa: PLC0415
        from ..models.transactional_request import TransactionalRequest # noqa: PLC0415
        from ..models.transactional_resend_state import TransactionalResendState # noqa: PLC0415
        from ..models.transactional_route_info import TransactionalRouteInfo # noqa: PLC0415
        from ..models.transactional_suppression_info import TransactionalSuppressionInfo # noqa: PLC0415
        from ..models.transactional_timeline_entry import TransactionalTimelineEntry # noqa: PLC0415
        from ..models.transactional_webhook_delivery import TransactionalWebhookDelivery # noqa: PLC0415
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        attachments = []
        _attachments = d.pop("attachments")
        for attachments_item_data in (_attachments):
            attachments_item = TransactionalAttachmentMeta.from_dict(attachments_item_data)



            attachments.append(attachments_item)


        attempts = []
        _attempts = d.pop("attempts")
        for attempts_item_data in (_attempts):
            attempts_item = TransactionalAttempt.from_dict(attempts_item_data)



            attempts.append(attempts_item)


        bcc = cast(list[str], d.pop("bcc"))


        can_remove_suppression = d.pop("canRemoveSuppression")

        cc = cast(list[str], d.pop("cc"))


        content = TransactionalContentState.from_dict(d.pop("content"))




        content_bytes = d.pop("contentBytes")

        deferrals = d.pop("deferrals")

        dropped = cast(list[str], d.pop("dropped"))


        email = d.pop("email")

        from_ = d.pop("from")

        id = d.pop("id")

        metadata = TransactionalMessageDetailMetadata.from_dict(d.pop("metadata"))




        raw = d.pop("raw")

        request = TransactionalRequest.from_dict(d.pop("request"))




        resend = TransactionalResendState.from_dict(d.pop("resend"))




        role_address = d.pop("roleAddress")

        route = TransactionalRouteInfo.from_dict(d.pop("route"))




        sensitive_sent = cast(list[str], d.pop("sensitiveSent"))


        size_bytes = d.pop("sizeBytes")

        soft_bounce_limit = d.pop("softBounceLimit")

        source = d.pop("source")

        status = TransactionalMessageDetailStatus(d.pop("status"))




        subject = d.pop("subject")

        tags = cast(list[str], d.pop("tags"))


        test_delivered = d.pop("testDelivered")

        timeline = []
        _timeline = d.pop("timeline")
        for timeline_item_data in (_timeline):
            timeline_item = TransactionalTimelineEntry.from_dict(timeline_item_data)



            timeline.append(timeline_item)


        tracked = d.pop("tracked")

        variables = d.pop("variables")

        webhooks = []
        _webhooks = d.pop("webhooks")
        for webhooks_item_data in (_webhooks):
            webhooks_item = TransactionalWebhookDelivery.from_dict(webhooks_item_data)



            webhooks.append(webhooks_item)


        _bounce = d.pop("bounce", UNSET)
        bounce: TransactionalBounce | Unset
        if isinstance(_bounce,  Unset):
            bounce = UNSET
        else:
            bounce = TransactionalBounce.from_dict(_bounce)




        category = d.pop("category", UNSET)

        category_kind = d.pop("categoryKind", UNSET)

        _clicked_at = d.pop("clickedAt", UNSET)
        clicked_at: datetime.datetime | Unset
        if isinstance(_clicked_at,  Unset):
            clicked_at = UNSET
        else:
            clicked_at = datetime.datetime.fromisoformat(_clicked_at)




        _contact = d.pop("contact", UNSET)
        contact: TransactionalRef | Unset
        if isinstance(_contact,  Unset):
            contact = UNSET
        else:
            contact = TransactionalRef.from_dict(_contact)




        contact_id = d.pop("contactId", UNSET)

        delivery_ms = d.pop("deliveryMs", UNSET)

        error = d.pop("error", UNSET)

        _failed_over = d.pop("failedOver", UNSET)
        failed_over: TransactionalFailover | Unset
        if isinstance(_failed_over,  Unset):
            failed_over = UNSET
        else:
            failed_over = TransactionalFailover.from_dict(_failed_over)




        idempotency_key = d.pop("idempotencyKey", UNSET)

        ip = d.pop("ip", UNSET)

        key = d.pop("key", UNSET)

        _key_ref = d.pop("keyRef", UNSET)
        key_ref: TransactionalRef | Unset
        if isinstance(_key_ref,  Unset):
            key_ref = UNSET
        else:
            key_ref = TransactionalRef.from_dict(_key_ref)




        _language_info = d.pop("languageInfo", UNSET)
        language_info: TransactionalMessageLanguage | Unset
        if isinstance(_language_info,  Unset):
            language_info = UNSET
        else:
            language_info = TransactionalMessageLanguage.from_dict(_language_info)




        name = d.pop("name", UNSET)

        _opened_at = d.pop("openedAt", UNSET)
        opened_at: datetime.datetime | Unset
        if isinstance(_opened_at,  Unset):
            opened_at = UNSET
        else:
            opened_at = datetime.datetime.fromisoformat(_opened_at)




        _provider = d.pop("provider", UNSET)
        provider: TransactionalRef | Unset
        if isinstance(_provider,  Unset):
            provider = UNSET
        else:
            provider = TransactionalRef.from_dict(_provider)




        provider_message_id = d.pop("providerMessageId", UNSET)

        _rejection = d.pop("rejection", UNSET)
        rejection: TransactionalRejection | Unset
        if isinstance(_rejection,  Unset):
            rejection = UNSET
        else:
            rejection = TransactionalRejection.from_dict(_rejection)




        reply_to = d.pop("replyTo", UNSET)

        resent_from = d.pop("resentFrom", UNSET)

        _send_at = d.pop("sendAt", UNSET)
        send_at: datetime.datetime | Unset
        if isinstance(_send_at,  Unset):
            send_at = UNSET
        else:
            send_at = datetime.datetime.fromisoformat(_send_at)




        sent_by = d.pop("sentBy", UNSET)

        soft_bounces = d.pop("softBounces", UNSET)

        _suppression = d.pop("suppression", UNSET)
        suppression: TransactionalSuppressionInfo | Unset
        if isinstance(_suppression,  Unset):
            suppression = UNSET
        else:
            suppression = TransactionalSuppressionInfo.from_dict(_suppression)




        _template = d.pop("template", UNSET)
        template: TransactionalLogTemplate | Unset
        if isinstance(_template,  Unset):
            template = UNSET
        else:
            template = TransactionalLogTemplate.from_dict(_template)




        transactional_message_detail = cls(
            at=at,
            attachments=attachments,
            attempts=attempts,
            bcc=bcc,
            can_remove_suppression=can_remove_suppression,
            cc=cc,
            content=content,
            content_bytes=content_bytes,
            deferrals=deferrals,
            dropped=dropped,
            email=email,
            from_=from_,
            id=id,
            metadata=metadata,
            raw=raw,
            request=request,
            resend=resend,
            role_address=role_address,
            route=route,
            sensitive_sent=sensitive_sent,
            size_bytes=size_bytes,
            soft_bounce_limit=soft_bounce_limit,
            source=source,
            status=status,
            subject=subject,
            tags=tags,
            test_delivered=test_delivered,
            timeline=timeline,
            tracked=tracked,
            variables=variables,
            webhooks=webhooks,
            bounce=bounce,
            category=category,
            category_kind=category_kind,
            clicked_at=clicked_at,
            contact=contact,
            contact_id=contact_id,
            delivery_ms=delivery_ms,
            error=error,
            failed_over=failed_over,
            idempotency_key=idempotency_key,
            ip=ip,
            key=key,
            key_ref=key_ref,
            language_info=language_info,
            name=name,
            opened_at=opened_at,
            provider=provider,
            provider_message_id=provider_message_id,
            rejection=rejection,
            reply_to=reply_to,
            resent_from=resent_from,
            send_at=send_at,
            sent_by=sent_by,
            soft_bounces=soft_bounces,
            suppression=suppression,
            template=template,
        )

        return transactional_message_detail

