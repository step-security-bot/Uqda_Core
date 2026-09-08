# UQDA documentation / توثيق عُقَد

UQDA is a separately maintained project derived from Yggdrasil: an encrypted,
self-organizing IPv6 overlay over existing network connections. It does not
replace your Internet connection, provide anonymity, or automatically route all
ordinary Internet traffic. See [upstream attribution](../NOTICE.md).

عُقَد مبنية على Yggdrasil، وليست اختراعًا منفصلًا لنفس التقنيات ولا إصدارًا
رسميًا صادرًا عن فريقهم. هذا الفهرس يرتب الشرح من الفكرة إلى التشغيل؛ المراجع
تصف الكود في الفرع الذي تقرؤه، وليس بالضرورة آخر ملف تثبيت نزلته.

## Start here / ابدأ هنا

| Your goal / المطلوب | Documentation / المرجع |
| --- | --- |
| Understand identity, packets, routing and encryption | [English project guide](PROJECT_GUIDE.md) / [الدليل العربي للشبكة](NETWORK_GUIDE_AR.md) |
| Install and operate Linux, macOS, Windows, BSD, containers or routers | [Platform commands in Arabic](NETWORK_GUIDE_AR.md#التثبيت-والتشغيل-حسب-النظام) / [English installation matrix](PROJECT_GUIDE.md#installation-by-operating-system) |
| Generate or edit persistent settings | [Configuration reference](configuration-reference.md) |
| Query a node, diagnose it or integrate a client | [Administration CLI and JSON API](admin-api.md) |
| Understand what differs from Yggdrasil | [المصدر والفروقات وحدود التوافق](UPSTREAM_COMPARISON_AR.md) |
| Migrate an old Windows installation | [Windows installation and trust](windows-installation.md) |
| Verify downloads | [Release verification](release-verification.md) / [Signing policy](CODE_SIGNING_POLICY.md) |
| Update or remove a node safely | [التحديث والإزالة](NETWORK_GUIDE_AR.md#التحديث-والإزالة) |
| Contribute or report security problems | [Contributing](../CONTRIBUTING.md) / [Security](../SECURITY.md) |

## A working node is more than a successful installation

1. Install the matching OS/CPU package; verify the downloaded release.
2. Preserve a unique private key in a persistent configuration.
3. Start the platform service, then query `getSelf` using `uqdactl`.
4. Establish an authorized direct peering or permitted LAN discovery.
5. Check `getPeers`, `getTun` and `doctor`; test the intended IPv6 application.

No peers means no usable remote path yet, not necessarily a broken installation.
An active peer link is also not proof that a destination application is listening
or that its firewall and group authentication permit your traffic.

## أسئلة أساسية

**هل تعطيني الشبكة إنترنت مجانيًا؟** لا؛ تحتاج وصلة موجودة بين النظراء، وقد
تكون إنترنت أو LAN أو شبكة خاصة. عنوان الشبكة المتراكبة لا يستبدل مزود الإنترنت.

**هل تشغّل كل تطبيقاتي تلقائيًا؟** التطبيقات القادرة على IPv6 تستطيع استخدام
عناوين الشبكة عبر TUN عندما يتوفر المسار ويسمح جدار الحماية. هذا ليس تحويلًا
تلقائيًا لكل اتصالات IPv4 أو شبكة VPN بخروج عام جاهز.

**هل أحتاج نظيرًا عامًا؟** ليس في شبكة محلية أو خاصة مترابطة. خارجها تحتاج
نظيرًا يمكن الوصول إليه. لا توجد قائمة نظراء عامة مضمونة التشغيل ضمن هذه الصفحات؛
احصل على URI ومفتاح موثوق من مشغّل النظير بدل نسخ عناوين أمثلة.

**هل اسم UQDA يجعلها شبكة معزولة عن Yggdrasil؟** لا. العزل يتعلق بالتوافق
والاتصالات وإعدادات المجموعة وجدار الحماية، وليس باسم الملف التنفيذي.

**هل تعني كلمة stable ضمان الأمان والتوافق مع كل جهاز؟** لا. راجع ملاحظات
الإصدار ونطاق اختبارات المنصة. لم يخضع المشروع لتدقيق أمني مستقل شامل.

## Keeping the documentation truthful

The configuration and API references link to their implementation. CI checks
local Markdown targets, platform facts and configuration-field coverage. These
checks detect drift; they do not certify every sentence or external service.
Old upstream blog posts are historical explanations, not current UQDA API specs.
